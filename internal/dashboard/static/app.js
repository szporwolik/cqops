// CQOps Live — two-state SSE dashboard with Leaflet map
(function(){
'use strict';

// ---- Debug: enable with ?debug=1 in the URL ----
var DEBUG=/[?&]debug=1(&|$)/.test(location.search);
function D(tag,msg,data){
  if(!DEBUG)return;
  var ts=new Date().toISOString().slice(11,23);
  if(data!==undefined){console.log('%c['+ts+'] %c'+tag+'%c '+msg,'color:#888','color:#D00032;font-weight:700','color:inherit',data)}
  else{console.log('%c['+ts+'] %c'+tag+'%c '+msg,'color:#888','color:#D00032;font-weight:700','color:inherit')}
}
if(DEBUG){D('init','debug mode ON — open console (F12) for SSE/event traces');console.log('%cAdd ?debug=1 to URL for these logs','color:#B45309')}

var $=function(id){return document.getElementById(id)};
var app=$('app'), hdLogo=$('hd-logo'), hdLogoBox=$('hd-logo-box'), hdTitle=$('hd-title'), hdSubtitle=$('hd-subtitle');
var hdClockLocal=$('hd-clock-local'), hdClockUtc=$('hd-clock-utc'), hdSSE=$('hd-sse-status');
var heroOverview=$('hero-overview'), heroHeadline=$('hero-headline'), heroSubline=$('hero-subline'), heroStatus=$('hero-status'), heroPromo=$('hero-promo');
var heroLabel=$('hero-label'),heroCall=$('hero-call'),heroBadges=$('hero-badges'),heroIdentity=$('hero-identity'),heroMeta=$('hero-meta');
var stationFields=$('station-fields'), statsFields=$('stats-fields'), operatorsFields=$('operators-fields'), topqsosFields=$('topqsos-fields');
var recentBody=$('recent-body');
var mapContainer=$('map-container');

var es=null, map=null;
var ownStationLat=null,ownStationLon=null;
var todayQsos=[], displayCfg={};

// ---- Clocks ----
function updateClocks(){
  var n=new Date();
  var local=n.toLocaleTimeString([],{hour:'2-digit',minute:'2-digit',second:'2-digit'});
  var utc=n.toISOString().slice(11,19).replace(/:/g,'')+'Z';
  hdClockLocal.textContent=local+' local '+utc;
  hdClockUtc.textContent='';
}
updateClocks();setInterval(updateClocks,1000);

// ---- State switching ----
function setState(active){
  var cls=active?'mode-active':'mode-overview';
  if(app.className!==cls){D('state','switch → '+cls);app.className=cls}
  if(active&&!todayQsos.length)updateMapFromToday()
}
function switchToOverview(){D('state','overview');setState(false);activeGrid=null;updateMapFromToday()}
function switchToActive(){D('state','active');setState(true)}

// ---- SSE ----
var sseReconnects=0;
function setSSEStatus(cls){hdSSE.className=cls;hdSSE.textContent=cls==='sse-connected'?'●':cls==='sse-connecting'?'◌':'○'}

function connectSSE(){
  if(es)es.close();setSSEStatus('sse-connecting');
  D('sse','connecting…');
  es=new EventSource('/api/events');
  es.addEventListener('snapshot',function(e){
    var s=JSON.parse(e.data).payload;
    D('sse','snapshot',{hasActive:!!(s.activeQso&&s.activeQso.call),today:s.today? s.today.length:0,recent:s.recent? s.recent.length:0,stats:s.stats});
    renderAll(s);
  });
  es.addEventListener('active_qso',function(e){var a=JSON.parse(e.data).payload;
    D('sse','active_qso',a||'<cleared>');
    if(a&&a.call){switchToActive();renderHero(a,lastPartner&&lastPartner.call===a.call?lastPartner:null)}
    else{switchToOverview();renderHero(null);lastActiveFlags={}}
  });
  es.addEventListener('qso_logged',function(e){var q=JSON.parse(e.data).payload;
    D('sse','qso_logged',q.call+' '+q.band+' '+q.mode);
    appendTodayQSO(q);prependRecentRow(q);updateMapFromToday();switchToOverview();renderHero(null);renderStats(null,todayQsos)
  });
  es.addEventListener('rig',function(e){var r=JSON.parse(e.data).payload;
    D('sse','rig',r.connected?(r.frequency||'?')+' '+(r.mode||''):'disconnected');
    updateStationField('Rig',r.connected?'<span class=\"status-on\">●</span> Connected':'<span class=\"status-off\">○</span> Disconnected');
    if(r.frequency)updateStationField('Frequency',r.frequency+' '+(r.mode||''));
  });
  es.addEventListener('wsjtx',function(e){var w=JSON.parse(e.data).payload;
    D('sse','wsjtx',w.connected?'online':'offline');
    updateStationField('WSJT-X',w.connected?'<span class=\"status-on\">●</span> Online':'<span class=\"status-off\">○</span> Offline')
  });
  es.addEventListener('stats',function(e){var s=JSON.parse(e.data).payload;
    D('sse','stats',{today:s.qsosToday,calls:s.uniqueCalls,rate:s.ratePerHour});
    renderStats(s,todayQsos)
  });
  es.addEventListener('recent_qsos',function(e){var r=JSON.parse(e.data).payload;
    D('sse','recent_qsos',r? r.length:0);
    renderRecentTable(r)
  });
  es.addEventListener('station',function(e){var s=JSON.parse(e.data).payload;
    D('sse','station',s.callsign+' '+s.locator);
    if(s.lat!=null&&s.lon!=null){ownStationLat=s.lat;ownStationLon=s.lon;updateMapFromToday()}renderStation(s)
  });
  es.addEventListener('operator',function(e){var o=JSON.parse(e.data).payload;
    D('sse','operator',o.callsign);
    updateStationField('Operator',o.callsign+(o.name?' ('+o.name+')':''))
  });
  es.addEventListener('logbook',function(e){var lb=JSON.parse(e.data).payload;
    D('sse','logbook',lb.name);
    updateStationField('Logbook',lb.name)
  });
  es.addEventListener('partner',function(e){var p=JSON.parse(e.data).payload;
    D('sse','partner',p? p.call+' '+(p.imageUrl?'📷':''):'<cleared>');
    lastPartner=p;
    if(p&&p.imageUrl){
      // Photo available — try showing it immediately even if the
      // active_qso event hasn't fired yet (the call is already in
      // the hero from a previous render or a race condition).
      var c=heroCall.textContent;
      if(c&&p.call&&c.toUpperCase()===p.call.toUpperCase())showHeroPhoto(p.imageUrl);
    }
    if(p){
      if(app.className==='mode-active'){var c=heroCall.textContent;if(c)renderHero({call:c},p)}
    }else{
      if(app.className==='mode-active'){hideHeroPhoto();heroIdentity.textContent='';heroMeta.textContent=''}
    }
  });
  es.addEventListener('today',function(e){
    var t=JSON.parse(e.data).payload;
    D('sse','today',t? t.length:0);
    // Only replace if the server has at least as many QSOs as we already
    // have — prevents the map from flickering down to 1 QSO right after
    // logging, before the server catches up.
    if(t&&t.length>=todayQsos.length){todayQsos=t;updateMapFromToday();renderStats(null,todayQsos)}
  });
  es.addEventListener('heartbeat',function(){D('sse','heartbeat')});
  es.onopen=function(){sseReconnects=0;setSSEStatus('sse-connected');D('sse','connected ✓')};
  es.onerror=function(){sseReconnects++;setSSEStatus('sse-disconnected');es.close();
    D('sse','error — reconnect #'+sseReconnects+' in 3s');
    setTimeout(function(){
      D('sse','fetching /api/snapshot fallback…');
      fetch('/api/snapshot').then(function(r){return r.json()}).then(renderAll).catch(function(){});connectSSE()
    },3000);
  };
}

// ---- Render all ----
function renderAll(snap){
  if(!snap){D('renderAll','null snapshot, skipping');return}
  D('renderAll','start',{hasStation:!!snap.station,hasActive:!!(snap.activeQso&&snap.activeQso.call),today:snap.today? snap.today.length:0,recent:snap.recent? snap.recent.length:0});
  displayCfg=snap.display||{};
  // Display config
  if(displayCfg.clubLogo){hdLogo.src=displayCfg.clubLogo;hdLogoBox.style.display=''}else{hdLogoBox.style.display='none'}
  if(displayCfg.header1){hdTitle.textContent=displayCfg.header1;heroHeadline.textContent=displayCfg.header1}
  else{hdTitle.textContent='CQOps Live';heroHeadline.textContent='CQOps Live'}
  hdSubtitle.textContent=displayCfg.header2||'Fast, portable ham radio logger';heroSubline.textContent=displayCfg.header2||'';
  // Marketing/PR line in hero when no custom header2 is configured.
  heroPromo.textContent=displayCfg.header2?'':'Powered by CQOps &mdash; cqops.com';
  // Station
  if(snap.station&&snap.station.lat&&snap.station.lon){
    ownStationLat=snap.station.lat;ownStationLon=snap.station.lon;
  }
  renderStation(snap.station,snap.operator,snap.logbook,snap.rig,snap.wsjtx);
  // Active QSO
  if(snap.activeQso&&snap.activeQso.call){switchToActive();renderHero(snap.activeQso,snap.partner)}
  else{switchToOverview();renderHero(null)}
  // Stats + recent
  renderStats(snap.stats,todayQsos);renderRecentTable(snap.recent);
  // Map
  D('renderAll','init map…');
  initMap(Object.assign({},displayCfg,{drawLines:displayCfg.drawLines!==false,maxLines:displayCfg.maxLines||250,highlightLastQSO:displayCfg.highlightLastQSO!==false,animateActivePath:!!displayCfg.animateActivePath}));
  // Init today QSOs — prefer today list, but fall back to recent
  // when today has fewer entries (midnight crossing, event just started).
  var tLen=snap.today? snap.today.length:0,rLen=snap.recent? snap.recent.length:0;
  if(tLen>=rLen&&tLen>0){
    D('renderAll','using snap.today ('+tLen+' QSOs)');
    todayQsos=snap.today.slice();updateMapFromToday();renderStats(null,todayQsos);
  }else if(rLen>0){
    D('renderAll','fallback to snap.recent ('+rLen+' QSOs)');
    todayQsos=snap.recent.slice();updateMapFromToday();
  }else{
    D('renderAll','no QSOs yet, map starts empty');
    todayQsos=[];updateMapFromToday();
  }
  // Refresh from /api/today in background for any QSOs saved after snapshot
  D('renderAll','fetching /api/today…');
  fetch('/api/today').then(function(r){return r.json()}).then(function(today){
    D('fetch','/api/today → '+(today? today.length:0)+' QSOs');
    if(today&&today.length&&today.length!==todayQsos.length){todayQsos=today;updateMapFromToday();renderStats(null,todayQsos)}
  }).catch(function(e){D('fetch','/api/today ERR',''+e)});
  // Footer: version + map attributions.
  if(snap.app&&snap.app.version){
    $('footer-text').innerHTML='CQOps Live v'+esc(snap.app.version)+' · <a href=\"https://cqops.com\" style=\"color:var(--accent)\">cqops.com</a>';
  }
  $('footer-attrib').innerHTML='Map: <a href=\"https://leafletjs.com\" target=\"_blank\" rel=\"noopener\">Leaflet</a> · Tiles: <a href=\"https://www.openstreetmap.org/copyright\" target=\"_blank\" rel=\"noopener\">&copy; OpenStreetMap contributors</a>';
  D('renderAll','done');
}

// Track the last active QSO + flags so they survive between SSE events.
var lastActiveFlags={},lastActiveQso=null,lastPartner=null;

// ---- Integrated active panel (hero + partner merged) ----
function renderHero(aq,p){
  if(!aq||!aq.call){heroCall.textContent='';heroBadges.innerHTML='';heroIdentity.textContent='';heroMeta.textContent='';hideHeroPhoto();lastActiveFlags={};lastActiveQso=null;D('hero','cleared');return}
  // Merge with cached active QSO: caller may pass {call:…} only (partner event).
  // Preserve band/mode/flags from cached full QSO when available.
  if(aq.band||aq.isDupe!==undefined){lastActiveQso=aq}
  else if(lastActiveQso&&lastActiveQso.call===aq.call){aq=lastActiveQso}
  else{lastActiveQso=aq}
  heroCall.textContent=aq.call;
  // Preserve dupe/new flags from aq if present, else use cached.
  if(aq.isDupe!==undefined)lastActiveFlags={isDupe:aq.isDupe,isNewDxcc:aq.isNewDxcc,isNewCall:aq.isNewCall};
  // Build badges using DOM API — guarantees separate elements, no merging.
  heroBadges.innerHTML='';
  function addBadge(text,cls){
    var s=document.createElement('span');s.className='badge'+(cls?' '+cls:'');s.textContent=text;heroBadges.appendChild(s);
  }
  if(aq.band)addBadge(aq.band);
  if(aq.mode)addBadge(aq.mode);
  if(lastActiveFlags.isDupe)addBadge('DUPE','dupe');
  else if(lastActiveFlags.isNewDxcc)addBadge('NEW DXCC','success');
  else if(lastActiveFlags.isNewCall)addBadge('NEW CALL','info');
  D('hero','render',{call:aq.call,band:aq.band,mode:aq.mode,dupe:aq.isDupe,newCall:aq.isNewCall,newDxcc:aq.isNewDxcc});
  // Photo
  if(p&&p.imageUrl){showHeroPhoto(p.imageUrl)}else{showHeroPlaceholder(p?p.call:aq.call)}
  // Identity line: merge partner data with form fields
  buildIdentityLine(aq,p);
  // Meta
  heroMeta.textContent=p&&p.source==='qrz'?'Source: QRZ.com lookup':'';
  // Map focus — debounced so QRZ data (more accurate grid) can arrive
  // before the fly animation starts. Avoids double-jump.
  // Priority: QSO form grid (may come from SOTA/POTA/WWFF reference
  // calculation or manual entry) beats QRZ partner grid.
  var mapGrid=aq.grid||(p&&p.grid)||'';
  if(mapGrid){
    scheduleMapFly(mapGrid,aq.call);
    drawActiveLine(mapGrid);
  }
}

// ---- Debounced map fly ----
var _flyTimer=null,_flyCall='';
function scheduleMapFly(grid,call){
  if(_flyTimer){clearTimeout(_flyTimer);_flyTimer=null}
  _flyCall=call||'';
  _flyTimer=setTimeout(function(){
    _flyTimer=null;_flyCall='';
    focusMapOnGrid(grid);
  },1800);
}

function buildIdentityLine(aq,p){
  // Merge partner data with form fields — partner wins when both present,
  // except for grid: the QSO form grid (manual entry or reference-derived)
  // beats the QRZ/country grid.
  var name=(p&&p.name)||(aq&&aq.name)||'';
  var qth=(p&&p.qth)||(aq&&aq.qth)||'';
  var country=(p&&p.country)||(aq&&aq.country)||'';
  var grid=(aq&&aq.grid)||(p&&p.grid)||'';
  var parts=[],dist='';
  if(name)parts.push(name);if(qth)parts.push(qth);
  if(country){var c=country,cont=guessContinent(c);if(cont)c+=' ('+cont+')';parts.push(c)}
  if(grid)parts.push(grid);
  // Distance from own station to partner grid
  if(grid&&ownStationLat!=null&&ownStationLon!=null){
    var ll=gridToLatLon(grid),km=haversineKm(ownStationLat,ownStationLon,ll[0],ll[1]),deg=bearingDeg(ownStationLat,ownStationLon,ll[0],ll[1]);
    var dirs=['N','NE','E','SE','S','SW','W','NW'];dist=Math.round(km)+' km '+dirs[Math.round(deg/45)%8];parts.push(dist);
  }
  heroIdentity.textContent=parts.join(' \u2022 ');
  return dist;
}

function showHeroPhoto(url){
  var img=$('hero-photo');img.style.display='';img.src=url;
  img.onclick=function(){var o=$('photo-overlay');$('photo-overlay-img').src=url;o.style.display='flex'};
  $('hero-placeholder').style.display='none';
}
function showHeroPlaceholder(call){$('hero-photo').style.display='none';$('hero-placeholder').style.display='flex';$('hero-placeholder-text').textContent=call||''}
function hideHeroPhoto(){$('hero-photo').style.display='none';$('hero-placeholder').style.display='none'}

// ---- Station panel ----
function renderStation(st,op,lb,rig,wsjtx){
  if(!st)return;op=op||{};lb=lb||{};rig=rig||{};wsjtx=wsjtx||{};
  var opText=op.callsign||'';if(op.name)opText+=' ('+op.name+')';
  var rigDot=rig.connected?'<span class=\"status-on\">●</span> Connected':'<span class=\"status-off\">○</span> Disconnected';
  var wsjtxDot=wsjtx.connected?'<span class=\"status-on\">●</span> Online':'<span class=\"status-off\">○</span> Offline';
  var rigFreq=rig.frequency?rig.frequency+(rig.mode?' '+rig.mode:''):'';
  stationFields.innerHTML=[
    ['Operator',opText||'—'],['Logbook',lb.name||'—'],['Locator',st.locator||'—'],
    ['Radio',st.radio||'—'],['Antenna',st.antenna||'—'],['Power',st.powerW?st.powerW+' W':'—'],
    ['Rig',rigDot],['WSJT-X',wsjtxDot],['Frequency',rigFreq]
  ].map(function(r){return'<dt>'+r[0]+'</dt><dd id=\"sf-'+r[0]+'\">'+r[1]+'</dd>'}).join('');
}
function updateStationField(key,val){var el=document.getElementById('sf-'+key);if(el)el.innerHTML=val}

// ---- Stats ----
function renderStats(st,todayBuf){
  if(!st)st={};
  // Browser-side stats from today buffer when server data is thin.
  var qsosToday=st.qsosToday||0,session=st.qsosSession||0;
  var longestKm=0;
  if((!qsosToday||!st.uniqueCalls)&&todayBuf&&todayBuf.length){
    var u={},calls={},dxcc={},grids={},bands={},modes={};
    todayBuf.forEach(function(q){
      if(q.call){calls[q.call.toUpperCase()]=1}
      if(q.country){var cn=q.country.replace(/^The\s+/,'').replace(/^Republic Of\s+/,'').replace(/^Federal Republic Of\s+/,'').trim().substring(0,28);dxcc[cn]=1}
      if(q.band)bands[q.band]=1;
      if(q.mode)modes[q.mode]=1;
      if(q.grid&&q.grid.length>=4)grids[q.grid.toUpperCase().substring(0,4)]=1;
    });
    qsosToday=todayBuf.length;
    st.qsosToday=qsosToday;
    st.uniqueCalls=Object.keys(calls).length;
    st.dxcc=Object.keys(dxcc).length;
    st.grids=Object.keys(grids).length;
    st.bands=Object.keys(bands).length;
    st.modes=Object.keys(modes).length;
    if(todayBuf.length>1){
      var times=todayBuf.map(function(q){return q.timeUtc?new Date(q.timeUtc).getTime():0}).filter(function(t){return t>0}).sort();
      if(times.length>1){var spanH=(times[times.length-1]-times[0])/3600000;if(spanH>0.05)st.ratePerHour=todayBuf.length/spanH}
    }
  }
  // Longest QSO distance — scan today buffer.
  if(ownStationLat!=null&&ownStationLon!=null&&todayQsos.length){
    todayQsos.forEach(function(q){
      var d=distKm(q.grid);if(d>longestKm)longestKm=d;
    });
  }
  statsFields.innerHTML=[['QSOs',qsosToday||0],['Session',session||0],['Unique calls',st.uniqueCalls||0],['DXCC',st.dxcc||0],['Grids',st.grids||0],['Bands',st.bands||0],['Modes',st.modes||0],['Longest',longestKm?Math.round(longestKm)+' km':'—'],['Rate',(st.ratePerHour||0).toFixed(1)+'/hr']].map(function(r){return'<dt>'+r[0]+'</dt><dd>'+r[1]+'</dd>'}).join('');
  renderTopOperators();
  renderTopQSOs();
}

// ---- Top Operators (by QSO count in today's buffer) ----
function renderTopOperators(){
  if(!todayQsos.length){operatorsFields.innerHTML='<dt style=\"color:var(--dim)\">—</dt>';return}
  var counts={};
  todayQsos.forEach(function(q){
    var op=(q.operator||'?').trim();if(!op)op='?';
    counts[op]=(counts[op]||0)+1;
  });
  var sorted=Object.keys(counts).sort(function(a,b){return counts[b]-counts[a]}).slice(0,9);
  operatorsFields.innerHTML=sorted.map(function(op,i){
    return'<dt>'+(i+1)+'.</dt><dd>'+esc(op)+' <span style=\"color:var(--dim);font-size:0.78rem\">'+counts[op]+'</span></dd>';
  }).join('')||'<dt style=\"color:var(--dim)\">—</dt>';
}

// ---- Top QSOs (by distance in today's buffer) ----
function renderTopQSOs(){
  if(!todayQsos.length||ownStationLat==null){topqsosFields.innerHTML='<dt style=\"color:var(--dim)\">—</dt>';return}
  var ranked=todayQsos.map(function(q){
    return{call:q.call||'?',grid:q.grid,band:q.band||'',mode:q.mode||'',km:distKm(q.grid)};
  }).filter(function(r){return r.km>0}).sort(function(a,b){return b.km-a.km}).slice(0,9);
  topqsosFields.innerHTML=ranked.map(function(r,i){
    return'<dt>'+(i+1)+'.</dt><dd><strong>'+esc(r.call)+'</strong> <span style="color:var(--dim);font-size:0.78rem">'+Math.round(r.km)+' km '+esc(r.band)+' '+esc(r.mode)+'</span></dd>';
  }).join('')||'<dt style=\"color:var(--dim)\">—</dt>';
}

// ---- Distance helpers ----
function distKm(grid){
  if(ownStationLat==null||ownStationLon==null||!grid||grid.length<4)return 0;
  var ll=gridToLatLon(grid);if(!ll[0])return 0;
  return haversineKm(ownStationLat,ownStationLon,ll[0],ll[1]);
}
function formatDistDir(grid){
  var km=distKm(grid);if(!km)return '—';
  var ll=gridToLatLon(grid);
  var deg=bearingDeg(ownStationLat,ownStationLon,ll[0],ll[1]);
  var dirs=['N','NE','E','SE','S','SW','W','NW'];
  return Math.round(km)+' km '+dirs[Math.round(deg/45)%8];
}

// ---- Recent QSOs table ----
function renderRecentTable(qsos){
  var list=qsos&&qsos.length?qsos.slice(0,20):[];
  if(!list.length){recentBody.innerHTML='<tr><td colspan=\"7\" style=\"color:var(--dim)\">No QSOs yet</td></tr>';return}
  recentBody.innerHTML=list.map(function(q){
    var utc=q.timeUtc?q.timeUtc.slice(11,16).replace(':','')+'Z':'';
    var ctry=(q.country||'').replace(/^The\s+/,'').replace(/^Republic Of\s+/,'').replace(/^Federal Republic Of\s+/,'').trim().substring(0,22);
    var dist=formatDistDir(q.grid);
    return'<tr><td>'+utc+'</td><td><strong>'+esc(q.call)+'</strong></td><td>'+(q.band||'')+'</td><td>'+(q.mode||'')+'</td><td>'+esc(q.rstSent||'')+'/'+esc(q.rstRcvd||'')+'</td><td title="'+(q.grid||'')+'">'+dist+'</td><td title="'+(q.country||'')+'">'+esc(ctry)+'</td></tr>';
  }).join('');
}
function prependRecentRow(q){
  var utc=q.timeUtc?q.timeUtc.slice(11,16).replace(':','')+'Z':'';
  var dist=formatDistDir(q.grid);
  var row=document.createElement('tr');row.className='new-row';
  row.innerHTML='<td>'+utc+'</td><td><strong>'+esc(q.call)+'</strong></td><td>'+(q.band||'')+'</td><td>'+(q.mode||'')+'</td><td>'+esc(q.rstSent||'')+'/'+esc(q.rstRcvd||'')+'</td><td title="'+(q.grid||'')+'">'+dist+'</td><td>'+(q.country||'')+'</td>';
  if(recentBody.firstChild)recentBody.insertBefore(row,recentBody.firstChild);else recentBody.appendChild(row);
  while(recentBody.children.length>20)recentBody.removeChild(recentBody.lastChild);
}

// ---- Today QSO buffer ----
function appendTodayQSO(q){todayQsos.unshift(q);if(todayQsos.length>500)todayQsos.length=500}

// ---- Map (Leaflet with great-circle paths) ----
var mapCfg={drawLines:true,maxLines:250,highlightLastQSO:true,animateActivePath:false};
var stationLayer=null,qsoMarkerLayer=null,qsoLineLayer=null,lastQsoLayer=null,activeQsoLayer=null;
var lastQso=null,activeGrid=null;

function initMap(cfg){
  if(map)return;
  if(cfg.drawLines!==undefined)mapCfg.drawLines=!!cfg.drawLines;
  if(cfg.maxLines)mapCfg.maxLines=cfg.maxLines;
  if(cfg.highlightLastQSO!==undefined)mapCfg.highlightLastQSO=!!cfg.highlightLastQSO;
  if(cfg.animateActivePath!==undefined)mapCfg.animateActivePath=!!cfg.animateActivePath;
  map=L.map('map-container',{zoomControl:true,attributionControl:false}).setView([51,10],3);
  L.tileLayer(cfg.mapTileUrl||'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png',{maxZoom:19,attribution:cfg.mapAttrib||'&copy; OpenStreetMap'}).addTo(map);
  // Layer groups — ordered bottom to top
  qsoLineLayer=L.layerGroup().addTo(map);     // older QSO lines (bottom)
  lastQsoLayer=L.layerGroup().addTo(map);      // last QSO line
  activeQsoLayer=L.layerGroup().addTo(map);    // active QSO line (top)
  qsoMarkerLayer=L.layerGroup().addTo(map);    // QSO markers
  stationLayer=L.layerGroup().addTo(map);      // station marker (always on top)
}

function updateMapFromToday(){
  if(!map){D('map','not initialized, skipping');return}
  D('map','update',{todayQsos:todayQsos.length,hasStation:!!(ownStationLat!=null&&ownStationLon!=null)});
  // Clear all layers
  stationLayer.clearLayers();qsoMarkerLayer.clearLayers();qsoLineLayer.clearLayers();lastQsoLayer.clearLayers();activeQsoLayer.clearLayers();
  var bounds=[],hasStation=false;

  // ---- Station marker ----
  if(ownStationLat!=null&&ownStationLon!=null){
    var sm=L.circleMarker([ownStationLat,ownStationLon],{radius:9,color:'#007A3D',fillColor:'#007A3D',fillOpacity:0.85,weight:2.5});
    sm.bindTooltip(displayCfg.header1||'Station',{permanent:true,direction:'top',offset:[0,-10]});
    stationLayer.addLayer(sm);bounds.push([ownStationLat,ownStationLon]);hasStation=true;
  }

  // ---- QSO markers + lines ----
  if(todayQsos.length&&mapCfg.drawLines&&hasStation){
    var maxLines=mapCfg.maxLines||250;
    var drawn=0,lastQsoIdx=-1;
    // Find last QSO with coordinates
    for(var i=0;i<todayQsos.length;i++){var q=todayQsos[i];var ll=getQsoLatLon(q);if(ll){lastQsoIdx=i;break}}

    todayQsos.forEach(function(q,i){
      var ll=getQsoLatLon(q);if(!ll)return;
      var lat=ll[0],lon=ll[1];
      // Marker
      var isLast=(i===lastQsoIdx&&mapCfg.highlightLastQSO),isActive=(activeGrid&&q.grid&&q.grid.toUpperCase()===activeGrid.toUpperCase());
      var mr=isActive?7:isLast?5.5:3.5;
      var mc=isActive?'#D00032':isLast?'#005BBB':'#005BBB';
      var mf=isActive?1:isLast?0.85:0.5;
      var mk=L.circleMarker([lat,lon],{radius:mr,color:mc,fillColor:mc,fillOpacity:mf,weight:isActive?3:1.5});
      var popup=(q.call||'')+'<br>'+(q.band||'')+' '+(q.mode||'')+'<br>'+(q.grid||'');
      if(q.timeUtc)popup+='<br>'+q.timeUtc.slice(11,16)+'Z';
      if(q.country)popup+='<br>'+q.country;
      mk.bindTooltip(q.call||'',{direction:'top'});mk.bindPopup(popup);
      qsoMarkerLayer.addLayer(mk);bounds.push([lat,lon]);

      // Lines — only if within limit
      if(drawn<maxLines){
        var pts=greatCirclePoints(ownStationLat,ownStationLon,lat,lon,32);
        if(isActive){
          // Active QSO: prominent dashed line with animated dash offset.
          var alOpt={color:'#D00032',weight:4,opacity:0.9,dashArray:'12 8',className:'active-path-anim'};
          activeQsoLayer.addLayer(L.polyline(pts,alOpt));
        }else if(isLast&&mapCfg.highlightLastQSO){
          // Last QSO: blue, thicker, more opaque than older lines.
          lastQsoLayer.addLayer(L.polyline(pts,{color:'#005BBB',weight:3,opacity:0.75}));
        }else{
          // Older QSO: subtle line
          qsoLineLayer.addLayer(L.polyline(pts,{color:'#005BBB',weight:1.2,opacity:0.28}));
        }
        drawn++;
      }
    });
  }else if(todayQsos.length&&!hasStation){
    // No station coords — still show markers without lines
    todayQsos.forEach(function(q){
      var ll=getQsoLatLon(q);if(!ll)return;
      var mk=L.circleMarker(ll,{radius:3.5,color:'#005BBB',fillColor:'#005BBB',fillOpacity:0.5,weight:1.5});
      mk.bindTooltip(q.call||'',{direction:'top'});
      qsoMarkerLayer.addLayer(mk);bounds.push(ll);
    });
  }

  // ---- Active line (drawn on top, outside today loop) ----
  if(activeGrid&&ownStationLat!==null){
    var al=gridToLatLon(activeGrid);if(al[0]){
      activeQsoLayer.addLayer(L.polyline(greatCirclePoints(ownStationLat,ownStationLon,al[0],al[1],48),{color:'#D00032',weight:4,opacity:0.9,dashArray:'12 8',className:'active-path-anim'}));
      // Partner location marker — pulsing dot at the far end of the active line.
      activeQsoLayer.addLayer(L.circleMarker(al,{radius:7,color:'#D00032',fillColor:'#D00032',fillOpacity:0.35,weight:2.5,className:'partner-dot'}));
      bounds.push(al);
    }
  }

  // Fit bounds — but don't override active-QSO focus set by focusMapOnGrid.
  if(bounds.length>1&&!activeGrid)map.flyToBounds(bounds,{padding:[30,30],maxZoom:12});
  else if(!hasStation)map.flyTo([51,10],2);
}

function getQsoLatLon(q){if(q.lat&&q.lon)return[q.lat,q.lon];if(q.grid){var ll=gridToLatLon(q.grid);if(ll[0])return ll}return null}

function focusMapOnGrid(grid){if(!map)return;var ll=gridToLatLon(grid);if(!ll[0])return;if(ownStationLat!=null&&ownStationLon!=null){map.flyToBounds([[ownStationLat,ownStationLon],ll],{padding:[50,50],maxZoom:10,duration:2.5})}else{map.flyTo(ll,6,{duration:2})}}
function drawActiveLine(grid){if(grid===activeGrid)return;activeGrid=grid;updateMapFromToday()}

// ---- Great-circle interpolation (browser-side, no plugin needed) ----
function greatCirclePoints(lat1,lon1,lat2,lon2,steps){
  steps=steps||32;
  var toRad=Math.PI/180,toDeg=180/Math.PI;
  // Convert to Cartesian on unit sphere
  var phi1=lat1*toRad,lam1=lon1*toRad,phi2=lat2*toRad,lam2=lon2*toRad;
  var x1=Math.cos(phi1)*Math.cos(lam1),y1=Math.cos(phi1)*Math.sin(lam1),z1=Math.sin(phi1);
  var x2=Math.cos(phi2)*Math.cos(lam2),y2=Math.cos(phi2)*Math.sin(lam2),z2=Math.sin(phi2);
  // Angular distance
  var dot=x1*x2+y1*y2+z1*z2;if(dot>1)dot=1;if(dot<-1)dot=-1;
  var omega=Math.acos(dot);
  if(omega<1e-9)return[[lat1,lon1],[lat2,lon2]]; // points are identical
  var sinO=Math.sin(omega);
  var pts=[];
  for(var i=0;i<=steps;i++){
    var t=i/steps;
    var a=Math.sin((1-t)*omega)/sinO,b=Math.sin(t*omega)/sinO;
    var x=a*x1+b*x2,y=a*y1+b*y2,z=a*z1+b*z2;
    var lat=Math.atan2(z,Math.sqrt(x*x+y*y))*toDeg,lon=Math.atan2(y,x)*toDeg;
    pts.push([lat,lon]);
  }
  return pts;
}

// ---- Math (browser-side, keeps Go lean) ----
function gridToLatLon(grid){
  grid=grid.toUpperCase().trim();if(grid.length<4)return[0,0];
  var lon=(grid.charCodeAt(0)-65)*20-180,lat=(grid.charCodeAt(1)-65)*10-90;
  lon+=(grid.charCodeAt(2)-48)*2;lat+=(grid.charCodeAt(3)-48)*1;
  if(grid.length>=6){
    // 5th char → subsquare longitude (5′ steps), 6th char → latitude (2.5′ steps).
    lon+=(grid.charCodeAt(4)-65)*(5/60);
    lat+=(grid.charCodeAt(5)-65)*(2.5/60);
    // Center of the 5′×2.5′ subsquare — matches locator.ToLatLon in Go.
    lon+=2.5/60;lat+=1.25/60;
  }else{
    // Center of the 2°×1° square (4-char grid).
    lon+=1;lat+=0.5;
  }
  return[lat,lon];
}
function haversineKm(lat1,lon1,lat2,lon2){
  var R=6371,dLat=(lat2-lat1)*Math.PI/180,dLon=(lon2-lon1)*Math.PI/180;
  var a=Math.sin(dLat/2)*Math.sin(dLat/2)+Math.cos(lat1*Math.PI/180)*Math.cos(lat2*Math.PI/180)*Math.sin(dLon/2)*Math.sin(dLon/2);
  return R*2*Math.atan2(Math.sqrt(a),Math.sqrt(1-a));
}
function bearingDeg(lat1,lon1,lat2,lon2){
  var dLon=(lon2-lon1)*Math.PI/180,y=Math.sin(dLon)*Math.cos(lat2*Math.PI/180),x=Math.cos(lat1*Math.PI/180)*Math.sin(lat2*Math.PI/180)-Math.sin(lat1*Math.PI/180)*Math.cos(lat2*Math.PI/180)*Math.cos(dLon);
  return(Math.atan2(y,x)*180/Math.PI+360)%360;
}
function guessContinent(c){
  if(!c)return'';var eu='Poland|Germany|England|France|Italy|Spain|Netherlands|Belgium|Sweden|Finland|Norway|Denmark|Austria|Czech Republic|Slovak Republic|Hungary|Switzerland|Ukraine|European Russia|Scotland|Wales|Ireland|Portugal|Greece|Romania|Bulgaria|Croatia|Slovenia|Serbia|Bosnia-Herzegovina|Lithuania|Latvia|Estonia|Belarus|Moldova|Luxembourg|Monaco'.split('|');
  var na='United States|Canada|Mexico|Cuba|Jamaica|Bahamas|Dominican Republic'.split('|');
  var sa='Brazil|Argentina|Chile|Peru|Colombia|Venezuela|Uruguay|Ecuador'.split('|');
  var as='Japan|China|India|South Korea|Thailand|Indonesia|Philippines|Asiatic Russia|Malaysia|Singapore|Vietnam|Taiwan'.split('|');
  var oc='Australia|New Zealand|Fiji|Papua New Guinea'.split('|');
  var af='South Africa|Kenya|Nigeria|Ghana|Egypt|Morocco|Tunisia|Algeria'.split('|');
  if(eu.indexOf(c)>=0)return'Europe';if(na.indexOf(c)>=0)return'North America';if(sa.indexOf(c)>=0)return'South America';
  if(as.indexOf(c)>=0)return'Asia';if(oc.indexOf(c)>=0)return'Oceania';if(af.indexOf(c)>=0)return'Africa';return'';
}
function esc(s){return(s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')}

// ---- Init ----
D('init','fetching /api/snapshot…');
fetch('/api/snapshot').then(function(r){return r.json()}).then(function(snap){
  D('fetch','/api/snapshot OK',{hasStation:!!snap.station,hasActive:!!(snap.activeQso&&snap.activeQso.call),today:snap.today? snap.today.length:0,recent:snap.recent? snap.recent.length:0});
  renderAll(snap);
}).catch(function(e){D('fetch','/api/snapshot ERR',''+e)});
connectSSE();

// ---- Auto-reload every 5 minutes (silent safety net) ----
// If SSE silently breaks or the tab goes stale, a periodic reload
// ensures the display stays fresh. Preserves ?debug=1 and restores
// the correct state from the server snapshot.
setTimeout(function(){
  D('reload','5-minute auto-reload triggered');
  location.reload();
},300000);

})();
