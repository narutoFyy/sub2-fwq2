import http.server, sys, os, re, base64, urllib.parse
OUT = sys.argv[1]
PAGE = b"""<!doctype html><meta charset=utf-8><body><pre id=o>working</pre><script>
(async()=>{const n=new URLSearchParams(location.search).get('n')||'out.wav';
let d=window.name||''; let via='name'; if(d.length<1000){d=location.hash.slice(1);via='hash'}
const r=await fetch('/b64/'+n,{method:'POST',body:d}); document.getElementById('o').textContent='via '+via+' len '+d.length+' -> '+await r.text(); window.name='';})();
</script>"""
class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200); self.send_header("Content-Type","text/html"); self.end_headers(); self.wfile.write(PAGE)
    def do_POST(self):
        name = re.sub(r"[^a-zA-Z0-9_.-]", "", self.path.split("/")[-1]) or "out.wav"
        data = self.rfile.read(int(self.headers.get("Content-Length", 0)))
        try: raw = base64.b64decode(data)
        except Exception as e: raw = b""
        if len(raw) > 1000: open(os.path.join(OUT, name), "wb").write(raw)
        self.send_response(200); self.end_headers(); self.wfile.write(f"saved {len(raw)} bytes".encode())
    def log_message(self, *a): pass
http.server.HTTPServer(("127.0.0.1", 38917), H).serve_forever()
