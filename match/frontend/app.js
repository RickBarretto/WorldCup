(function(){
  const logEl = document.getElementById('log')
  const playerIdEl = document.getElementById('playerId')
  const connectBtn = document.getElementById('connectBtn')
  const disconnectBtn = document.getElementById('disconnectBtn')
  const cardsTbody = document.getElementById('cardsTbody')
  const randomizeBtn = document.getElementById('randomize')
  const playBtn = document.getElementById('playBtn')

  let ws = null

  function log(...args){
    logEl.textContent += args.map(a=>typeof a==='object'?JSON.stringify(a,null,2):String(a)).join(' ') + '\n'
    logEl.scrollTop = logEl.scrollHeight
  }

  function makeRow(i, c){
    const tr = document.createElement('tr')
    tr.innerHTML = `<td>${i+1}</td>` +
      `<td><input data-field="id" value="${escapeHtml(c.id||('c'+(i+1)))}"></td>` +
      `<td><input data-field="name" value="${escapeHtml(c.name||('Card '+(i+1)))}"></td>` +
      `<td><input data-field="power" type="number" value="${c.power||(i+1)}"></td>`
    return tr
  }

  function escapeHtml(s){ return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;') }

  function loadRows(cards){
    cardsTbody.innerHTML = ''
    for(let i=0;i<5;i++){
      const c = cards[i] || {id:'c'+(i+1),name:'Card '+(i+1),power:i+1}
      cardsTbody.appendChild(makeRow(i,c))
    }
  }

  function getCards(){
    const rows = cardsTbody.querySelectorAll('tr')
    const out = []
    rows.forEach(r=>{
      const id = r.querySelector('input[data-field=id]').value
      const name = r.querySelector('input[data-field=name]').value
      const power = parseInt(r.querySelector('input[data-field=power]').value,10)||0
      out.push({id,name,power})
    })
    return out
  }

  function randomize(){
    const cards = []
    for(let i=0;i<5;i++){
      cards.push({id:'r'+Math.random().toString(36).slice(2,8), name:'R'+(i+1), power: Math.floor(Math.random()*6)+1})
    }
    loadRows(cards)
  }

  connectBtn.addEventListener('click', ()=>{
    const pid = playerIdEl.value || ('player-'+Math.floor(Math.random()*1000))
    const url = (location.protocol==='https:'? 'wss://' : 'ws://') + location.host + '/ws?player_id=' + encodeURIComponent(pid)
    log('Connecting WS to', url)
    ws = new WebSocket(url)
    ws.onopen = ()=>{ log('WS open'); connectBtn.disabled = true; disconnectBtn.disabled = false }
    ws.onmessage = (ev)=>{ log('WS msg:', JSON.parse(ev.data)) }
    ws.onclose = ()=>{ log('WS closed'); connectBtn.disabled = false; disconnectBtn.disabled = true }
    ws.onerror = (e)=>{ log('WS error', e) }
  })

  disconnectBtn.addEventListener('click', ()=>{
    if(ws){ ws.close(); ws = null }
  })

  randomizeBtn.addEventListener('click', randomize)

  playBtn.addEventListener('click', async ()=>{
    const pid = playerIdEl.value || ('player-'+Math.floor(Math.random()*1000))
    const cards = getCards()
    if(cards.length !== 5){ alert('need 5 cards'); return }
    const body = { player_id: pid, cards }
    log('POST /play', body)
    try{
      const res = await fetch('/play', { method:'POST', headers:{'content-type':'application/json'}, body: JSON.stringify(body) })
      if(res.status === 202){ const txt = await res.text(); log('Queued:', txt); return }
      const j = await res.json()
      log('Play response:', j)
    }catch(err){ log('play error', err) }
  })

  // init
  loadRows([])
  randomize()

})();
