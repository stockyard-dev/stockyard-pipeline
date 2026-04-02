package server
import "net/http"
func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashHTML))
}
const dashHTML = `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Pipeline</title>
<style>:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#c45d2c;--rl:#e8753a;--leather:#a0845c;--ll:#c4a87a;--cream:#f0e6d3;--cd:#bfb5a3;--cm:#7a7060;--gold:#d4a843;--green:#4a9e5c;--red:#c44040;--mono:'JetBrains Mono',Consolas,monospace;--serif:'Libre Baskerville',Georgia,serif}*{margin:0;padding:0;box-sizing:border-box}body{background:var(--bg);color:var(--cream);font-family:var(--mono);font-size:13px;line-height:1.6}.hdr{padding:.6rem 1.2rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center}.hdr h1{font-family:var(--serif);font-size:1rem}.hdr h1 span{color:var(--rl)}.main{max-width:900px;margin:0 auto;padding:1rem 1.2rem}.btn{font-family:var(--mono);font-size:.68rem;padding:.3rem .6rem;border:1px solid;cursor:pointer;background:transparent;transition:.15s}.btn-p{border-color:var(--rust);color:var(--rl)}.btn-p:hover{background:var(--rust);color:var(--cream)}.btn-d{border-color:var(--bg3);color:var(--cm)}.btn-d:hover{border-color:var(--red);color:var(--red)}.btn-s{border-color:var(--green);color:var(--green)}.btn-s:hover{background:var(--green);color:var(--bg)}.card{background:var(--bg2);border:1px solid var(--bg3);padding:.7rem;margin-bottom:.4rem;cursor:pointer;transition:.1s}.card:hover{background:var(--bg3)}.card h3{font-size:.8rem;margin-bottom:.2rem}.card-meta{font-size:.65rem;color:var(--cm);display:flex;gap:.7rem}.st-success{color:var(--green)}.st-failed{color:var(--red)}.st-running{color:var(--gold)}.step-pill{display:inline-block;font-size:.6rem;padding:.1rem .3rem;background:var(--bg3);color:var(--ll);border-radius:2px;margin-right:.2rem}.empty{text-align:center;padding:2rem;color:var(--cm);font-style:italic;font-family:var(--serif)}.modal-bg{position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,.65);display:flex;align-items:center;justify-content:center;z-index:100}.modal{background:var(--bg2);border:1px solid var(--bg3);padding:1.5rem;width:95%;max-width:600px;max-height:90vh;overflow-y:auto}.modal h2{font-family:var(--serif);font-size:.95rem;margin-bottom:1rem}label.fl{display:block;font-size:.65rem;color:var(--leather);text-transform:uppercase;letter-spacing:1px;margin-bottom:.2rem;margin-top:.5rem}input[type=text],textarea,select{background:var(--bg);border:1px solid var(--bg3);color:var(--cream);padding:.35rem .5rem;font-family:var(--mono);font-size:.78rem;width:100%;outline:none}textarea{resize:vertical;min-height:60px}.run-row{display:flex;align-items:center;gap:.5rem;padding:.3rem .5rem;border-bottom:1px solid var(--bg3);font-size:.72rem}</style>
<link href="https://fonts.googleapis.com/css2?family=Libre+Baskerville:ital@0;1&family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet">
</head><body><div class="hdr"><h1><span>Pipeline</span></h1><button class="btn btn-p" onclick="showNew()">+ Pipeline</button></div>
<div class="main"><div id="list"></div><div id="detail" style="display:none;margin-top:1rem"></div></div><div id="modal"></div>
<script>
let pipelines=[],cur=null;
async function api(u,o){return(await fetch(u,o)).json()}
function esc(s){return String(s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;')}
function timeAgo(d){if(!d)return'never';const s=Math.floor((Date.now()-new Date(d))/1e3);if(s<60)return s+'s ago';if(s<3600)return Math.floor(s/60)+'m ago';return Math.floor(s/3600)+'h ago'}
async function init(){const d=await api('/api/pipelines');pipelines=d.pipelines||[];render()}
function render(){
  document.getElementById('list').innerHTML=pipelines.length?pipelines.map(p=>{
    const steps=(p.steps||[]).map(s=>'<span class="step-pill">'+esc(s.name)+'</span>').join('');
    return'<div class="card" onclick="open_(\''+p.id+'\')"><h3>'+esc(p.name)+'</h3>'+
    '<div class="card-meta"><span>'+(p.steps||[]).length+' steps</span><span>'+p.run_count+' runs</span>'+
    '<span class="st-'+(p.last_status||'')+'">'+esc(p.last_status||'no runs')+'</span>'+
    '<span>'+timeAgo(p.last_run)+'</span></div>'+
    '<div style="margin-top:.3rem">'+steps+'</div></div>'}).join(''):'<div class="empty">No pipelines yet.</div>'
}
async function open_(id){
  cur=id;const[p,rd]=await Promise.all([api('/api/pipelines/'+id),api('/api/pipelines/'+id+'/runs')]);
  const runs=(rd.runs||[]).map(r=>'<div class="run-row"><span class="st-'+r.status+'">'+r.status+'</span><span>'+r.duration_ms+'ms</span><span style="color:var(--cm)">'+timeAgo(r.started_at)+'</span></div>').join('');
  document.getElementById('detail').style.display='block';
  document.getElementById('detail').innerHTML='<div style="display:flex;justify-content:space-between;margin-bottom:.5rem"><span style="font-size:.75rem;color:var(--leather)">'+esc(p.name)+'</span><div style="display:flex;gap:.3rem"><button class="btn btn-s" onclick="runP(\''+id+'\')">▶ Run</button><button class="btn btn-d" onclick="if(confirm(\'Delete?\'))delP(\''+id+'\')">Del</button></div></div>'+
  '<div style="font-size:.7rem;color:var(--leather);margin-bottom:.3rem">Steps</div>'+(p.steps||[]).map(s=>'<div style="padding:.2rem .5rem;border-bottom:1px solid var(--bg3);font-size:.72rem">'+esc(s.name)+' <span style="color:var(--cm)">'+esc(s.type)+'</span></div>').join('')+
  '<div style="font-size:.7rem;color:var(--leather);margin:1rem 0 .3rem">Runs</div>'+(runs||'<div style="color:var(--cm);font-size:.72rem">No runs yet.</div>')
}
async function runP(id){await api('/api/pipelines/'+id+'/run',{method:'POST'});open_(id);init()}
async function delP(id){await api('/api/pipelines/'+id,{method:'DELETE'});cur=null;document.getElementById('detail').style.display='none';init()}
function showNew(){
  document.getElementById('modal').innerHTML='<div class="modal-bg" onclick="if(event.target===this)closeModal()"><div class="modal"><h2>New Pipeline</h2>'+
  '<label class="fl">Name</label><input type="text" id="np-name">'+
  '<label class="fl">Description</label><input type="text" id="np-desc">'+
  '<label class="fl">Steps (JSON array)</label><textarea id="np-steps" rows="4" placeholder=\'[{"name":"extract","type":"http"},{"name":"transform","type":"script"}]\'></textarea>'+
  '<label class="fl">Schedule (cron)</label><input type="text" id="np-sched" placeholder="0 * * * *">'+
  '<div style="display:flex;gap:.5rem;margin-top:1rem"><button class="btn btn-p" onclick="saveNew()">Create</button><button class="btn btn-d" onclick="closeModal()">Cancel</button></div></div></div>'
}
async function saveNew(){
  let steps=[];try{steps=JSON.parse(document.getElementById('np-steps').value||'[]')}catch(e){alert('Invalid JSON');return}
  const body={name:document.getElementById('np-name').value,description:document.getElementById('np-desc').value,steps,schedule:document.getElementById('np-sched').value};
  if(!body.name){alert('Name required');return}
  await api('/api/pipelines',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});closeModal();init()
}
function closeModal(){document.getElementById('modal').innerHTML=''}
init()
</script></body></html>`
