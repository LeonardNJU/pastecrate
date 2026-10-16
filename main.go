package main

import (
  "encoding/json"
  "fmt"
  "html/template"
  "io"
  "log"
  "mime"
  "net/http"
  "net/url"
  "os"
  "path/filepath"
  "regexp"
  "strings"
  "sync"
  "time"
)

const (
  dataDir = "/data"
  ttl = 72 * time.Hour
  maxBody = 100 << 20 // 100 MiB total
  maxText = 2 << 20
)

var (
  codeRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{1,63}$`)
  mu sync.Mutex
  reserved = map[string]bool{"create":true,"healthz":true,"favicon.ico":true,"robots.txt":true}
)

type FileMeta struct { Stored, Name string; Size int64; Type string }
type Meta struct { Code string `json:"code"`; Text string `json:"text"`; CreatedAt time.Time `json:"created_at"`; ExpiresAt time.Time `json:"expires_at"`; Files []FileMeta `json:"files"` }

type PageData struct { Exists bool; Code, Text string; Files []FileMeta; Expires string; Error string }

var page = template.Must(template.New("p").Parse(`<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{if .Code}}{{.Code}} - {{end}}PasteCrate</title><style>
:root{color-scheme:light dark;--bg:#f7f7f8;--card:#fff;--fg:#18181b;--muted:#71717a;--line:#e4e4e7;--acc:#5b5bd6;--danger:#dc2626}@media(prefers-color-scheme:dark){:root{--bg:#111113;--card:#18181b;--fg:#f4f4f5;--muted:#a1a1aa;--line:#303036;--acc:#7c7cf0}}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--fg);font:15px/1.5 system-ui,-apple-system,Segoe UI,sans-serif}.wrap{max-width:900px;margin:auto;padding:28px 18px}.top{display:flex;justify-content:space-between;align-items:center;margin-bottom:18px}.brand{font-weight:750;font-size:22px}.card{background:var(--card);border:1px solid var(--line);border-radius:14px;padding:18px;margin-bottom:14px}.code{font:600 18px ui-monospace,SFMono-Regular,Menlo,monospace}input,textarea{width:100%;border:1px solid var(--line);background:transparent;color:var(--fg);border-radius:9px;padding:10px 12px;font:inherit}textarea{min-height:280px;resize:vertical;font-family:ui-monospace,SFMono-Regular,Menlo,monospace}.drop{border:1px dashed var(--line);border-radius:10px;padding:14px;margin-top:12px}.row{display:flex;gap:10px;align-items:center}.row>*{flex:1}.actions{display:flex;gap:8px;justify-content:flex-end;margin-top:14px}button,.btn{border:0;border-radius:9px;padding:9px 14px;background:var(--acc);color:#fff;text-decoration:none;font-weight:650;cursor:pointer}.secondary{background:transparent;color:var(--fg);border:1px solid var(--line)}.danger{background:var(--danger)}.muted{color:var(--muted);font-size:13px}.files{display:grid;gap:8px}.file{display:flex;justify-content:space-between;gap:12px;align-items:center;padding:10px;border:1px solid var(--line);border-radius:9px}.file a{color:inherit;text-decoration:none;overflow-wrap:anywhere}.error{color:#dc2626}.viewtext{white-space:pre-wrap;overflow-wrap:anywhere;font:14px/1.55 ui-monospace,SFMono-Regular,Menlo,monospace;margin:0}
#upload-feedback{margin-top:14px}#upload-feedback progress{width:100%;height:12px;accent-color:var(--acc)}.progress-meta{display:flex;align-items:center;justify-content:space-between;gap:12px;margin-bottom:7px;font-size:13px}.progress-meta span:last-child{font-variant-numeric:tabular-nums;white-space:nowrap}.upload-error{color:var(--danger)}button:disabled{opacity:.6;cursor:wait}
@media(max-width:600px){.row{flex-direction:column}.actions{flex-wrap:wrap}.actions>*{flex:1}}
</style></head><body><div class="wrap"><div class="top"><div class="brand">PasteCrate</div>{{if .Exists}}<a class="btn secondary" href="/">New</a>{{end}}</div>
{{if .Error}}<div class="card error">{{.Error}}</div>{{end}}
{{if .Exists}}
<div class="card"><div class="code">/{{.Code}}</div><div class="muted">Expires {{.Expires}} · Anyone with this code can view and delete it.</div></div>
{{if .Text}}<div class="card"><pre id="txt" class="viewtext">{{.Text}}</pre><div class="actions"><button class="secondary" onclick="navigator.clipboard.writeText(document.getElementById('txt').innerText).then(()=>this.innerText='Copied')">Copy text</button></div></div>{{end}}
{{if .Files}}<div class="card"><div class="files">{{range .Files}}<div class="file"><a href="/{{$.Code}}/file/{{urlquery .Name}}">{{.Name}}</a><span class="muted">{{.Size}} bytes</span></div>{{end}}</div></div>{{end}}
<div class="actions"><form method="post" action="/{{.Code}}/delete" onsubmit="return confirm('Delete this paste now?')"><button class="danger" type="submit">Delete</button></form></div>
{{else}}
<div class="card"><form id="create-form" method="post" action="/create" enctype="multipart/form-data"><div class="row"><input name="code" required minlength="2" maxlength="64" pattern="[A-Za-z0-9][A-Za-z0-9_-]{1,63}" placeholder="Code, e.g. lab42"><button id="create-btn" type="submit">Create</button></div><p class="muted">Open later at /&lt;code&gt; on this server. Public by design. Auto-deletes after 3 days.</p><textarea name="text" maxlength="2097152" placeholder="Paste text here (optional)"></textarea><div class="drop"><input type="file" name="files" multiple><div class="muted">Optional files · any type · total request up to 100 MiB</div></div>
<div id="upload-feedback" hidden aria-live="polite"><div class="progress-meta"><span id="upload-status"></span><span id="upload-percent"></span></div><progress id="upload-progress" value="0" max="100" hidden></progress></div></form></div>
{{end}}
</div><script>
(function(){
  const form = document.getElementById("create-form");
  if (!form || !window.XMLHttpRequest || !window.FormData) return;
  const btn = document.getElementById("create-btn");
  const feedback = document.getElementById("upload-feedback");
  const status = document.getElementById("upload-status");
  const percent = document.getElementById("upload-percent");
  const bar = document.getElementById("upload-progress");
  const fileInput = form.querySelector('input[name="files"]');
  const MiB = 1024 * 1024;
  let inFlight = false;
  function formatSize(n){return (n / MiB).toFixed(1) + " MiB";}
  function resetState(){
    inFlight = false;
    btn.disabled = false;
    btn.textContent = "Create";
  }
  function fail(message){
    resetState();
    feedback.hidden = false;
    status.textContent = message;
    status.classList.add("upload-error");
    percent.textContent = "";
    bar.hidden = true;
  }
  form.addEventListener("submit", function(event){
    if (inFlight) { event.preventDefault(); return; }
    const files = fileInput ? Array.from(fileInput.files) : [];
    const selectedBytes = files.reduce(function(sum,file){return sum + file.size;},0);
    if (selectedBytes >= 100 * MiB) {
      event.preventDefault();
      fail("Files exceed the 100 MiB request limit. Choose smaller files.");
      return;
    }
    event.preventDefault();
    const showProgress = selectedBytes > 10 * MiB;
    const xhr = new XMLHttpRequest();
    inFlight = true;
    btn.disabled = true;
    btn.textContent = "Uploading...";
    feedback.hidden = false;
    status.classList.remove("upload-error");
    status.textContent = showProgress ? "Starting upload..." : "Submitting...";
    percent.textContent = "";
    bar.hidden = !showProgress;
    bar.value = 0;
    xhr.open("POST", form.action, true);
    xhr.responseType = "json";
    xhr.setRequestHeader("X-Requested-With", "XMLHttpRequest");
    xhr.upload.addEventListener("progress", function(e){
      if (!showProgress || !e.lengthComputable) return;
      const fraction = e.total ? e.loaded / e.total : 0;
      const value = Math.min(100,Math.round(fraction*100));
      bar.value = value;
      percent.textContent = value + "% · " + formatSize(e.loaded) + " / " + formatSize(e.total);
      status.textContent = value === 100 ? "Upload sent. Saving on server..." : "Uploading...";
    });
    xhr.upload.addEventListener("load",function(){
      status.textContent = "Upload sent. Saving on server...";
    });
    xhr.addEventListener("load", function(){
      const data = xhr.response;
      if(xhr.status >= 200 && xhr.status < 300 && data && data.ok && typeof data.url === "string" && data.url.charAt(0) === "/"){
        status.textContent = "Saved. Opening link...";
        window.location.assign(data.url);
      } else {
        fail((data && data.error) || ("Upload failed (HTTP " + xhr.status + "). Please retry."));
      }
    });
    xhr.addEventListener("error", function(){fail("Network error during upload. Please retry.");});
    xhr.addEventListener("abort", function(){fail("Upload interrupted. Please retry.");});
    xhr.send(new FormData(form));
  });
})();
</script></body></html>`))

func codeDir(code string) string { return filepath.Join(dataDir, code) }
func metaPath(code string) string { return filepath.Join(codeDir(code), "meta.json") }
func validCode(code string) bool { return codeRE.MatchString(code) && !reserved[strings.ToLower(code)] }

func load(code string) (*Meta,error) {
  b,err:=os.ReadFile(metaPath(code)); if err!=nil{return nil,err}
  var m Meta; if err=json.Unmarshal(b,&m); err!=nil{return nil,err}
  if time.Now().After(m.ExpiresAt){ _=os.RemoveAll(codeDir(code)); return nil,os.ErrNotExist }
  return &m,nil
}
func save(m *Meta) error { b,_:=json.MarshalIndent(m,"","  "); return os.WriteFile(metaPath(m.Code),b,0640) }
func render(w http.ResponseWriter,d PageData){ w.Header().Set("Cache-Control","no-store"); if err:=page.Execute(w,d);err!=nil{log.Print(err)} }

func root(w http.ResponseWriter,r *http.Request){
  if r.URL.Path!="/" { http.NotFound(w,r); return }
  render(w,PageData{})
}
func createResponse(w http.ResponseWriter, r *http.Request, code, message string, status int) {
  if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.Header().Set("Cache-Control", "no-store")
    if message != "" {
      w.WriteHeader(status)
      _ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": message})
    } else {
      _ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "url": "/" + url.PathEscape(code)})
    }
    return
  }
  if message != "" {
    if status >= 500 { http.Error(w, message, status) } else { render(w, PageData{Error:message}) }
    return
  }
  http.Redirect(w, r, "/" + url.PathEscape(code), http.StatusSeeOther)
}

func create(w http.ResponseWriter,r *http.Request){
  if r.Method!=http.MethodPost { http.Error(w,"method",405); return }
  r.Body=http.MaxBytesReader(w,r.Body,maxBody)
  if err:=r.ParseMultipartForm(8 << 20); err!=nil { createResponse(w,r,"","Upload too large or invalid.",413); return }
  code:=strings.TrimSpace(r.FormValue("code")); if !validCode(code){ createResponse(w,r,"","Invalid code. Use 2–64 letters, numbers, - or _.",400); return }
  text:=r.FormValue("text"); if len(text)>maxText { createResponse(w,r,"","Text is too large.",413); return }
  mu.Lock(); defer mu.Unlock()
  if _,err:=load(code); err==nil { createResponse(w,r,"","That code already exists. Pick another one.",409); return }
  dir:=codeDir(code); if err:=os.MkdirAll(filepath.Join(dir,"files"),0750);err!=nil{createResponse(w,r,"","Storage error. Please retry.",500);return}
  m:=&Meta{Code:code,Text:text,CreatedAt:time.Now().UTC(),ExpiresAt:time.Now().UTC().Add(ttl)}
  for _,fh:=range r.MultipartForm.File["files"]{
    src,err:=fh.Open(); if err!=nil{continue}
    clean:=filepath.Base(fh.Filename); if clean=="."||clean==""{src.Close();continue}
    stored:=fmt.Sprintf("%d-%s",len(m.Files),clean)
    dst,err:=os.Create(filepath.Join(dir,"files",stored)); if err!=nil{src.Close();continue}
    n,err:=io.Copy(dst,src); src.Close(); dst.Close(); if err!=nil{continue}
    typ:=fh.Header.Get("Content-Type"); if typ==""{typ=mime.TypeByExtension(filepath.Ext(clean))}
    m.Files=append(m.Files,FileMeta{Stored:stored,Name:clean,Size:n,Type:typ})
  }
  if strings.TrimSpace(text)=="" && len(m.Files)==0 { _=os.RemoveAll(dir); createResponse(w,r,"","Add some text or at least one file.",400); return }
  if err:=save(m);err!=nil{_ = os.RemoveAll(dir);createResponse(w,r,"","Storage error. Please retry.",500);return}
  createResponse(w,r,code,"",0)
}
func view(w http.ResponseWriter,r *http.Request){
  parts:=strings.Split(strings.Trim(r.URL.Path,"/"),"/"); if len(parts)==0||!validCode(parts[0]){http.NotFound(w,r);return}; code:=parts[0]
  m,err:=load(code); if err!=nil{http.NotFound(w,r);return}
  if len(parts)==1 { render(w,PageData{Exists:true,Code:code,Text:m.Text,Files:m.Files,Expires:m.ExpiresAt.Local().Format("Jan 2, 15:04")}); return }
  if len(parts)==3 && parts[1]=="file" { name,_:=url.PathUnescape(parts[2]); for _,f:=range m.Files{if f.Name==name{w.Header().Set("Content-Disposition",fmt.Sprintf("attachment; filename*=UTF-8''%s",url.PathEscape(f.Name))); if f.Type!=""{w.Header().Set("Content-Type",f.Type)}; http.ServeFile(w,r,filepath.Join(codeDir(code),"files",f.Stored)); return}} }
  if len(parts)==2 && parts[1]=="delete" && r.Method==http.MethodPost { _=os.RemoveAll(codeDir(code)); http.Redirect(w,r,"/",http.StatusSeeOther); return }
  http.NotFound(w,r)
}
func cleanup(){ for{ entries,_:=os.ReadDir(dataDir); for _,e:=range entries{if e.IsDir(){_,_=load(e.Name())}}; time.Sleep(15*time.Minute)} }
func main(){ _=os.MkdirAll(dataDir,0750); go cleanup(); http.HandleFunc("/healthz",func(w http.ResponseWriter,r *http.Request){io.WriteString(w,"OK")}); http.HandleFunc("/create",create); http.HandleFunc("/",func(w http.ResponseWriter,r *http.Request){if r.URL.Path=="/"{root(w,r)}else{view(w,r)}}); log.Println("pastecrate listening on :8080"); log.Fatal(http.ListenAndServe(":8080",nil)) }
