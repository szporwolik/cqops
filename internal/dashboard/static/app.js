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
var stationFields=$('station-fields'), statsFields=$('stats-fields'), topqsosFields=$('topqsos-fields');
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
  hdClockLocal.textContent='Local '+local+' \u00b7 UTC '+utc;
  hdClockUtc.textContent='';
}
updateClocks();setInterval(updateClocks,1000);

// ---- State switching ----
function setState(active){
  var add=active?'mode-active':'mode-overview',rm=active?'mode-overview':'mode-active';
  if(!app.classList.contains(add)){
    app.classList.remove(rm);app.classList.add(add);if(map)map.invalidateSize();
    D('state','switch → '+add);
  }
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
    appendTodayQSO(q);prependRecentRow(q);updateMapFromToday();switchToOverview();renderHero(null);renderStats(null,todayQsos);showQsoToast(q);playPing()
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
    if(s.lat!=null&&s.lon!=null){ownStationLat=s.lat;ownStationLon=s.lon;initLocalMap(s.lat,s.lon);recentreLocalMap(s.lat,s.lon);updateAprsCircle(s.lat,s.lon,s.aprsRadiusKm||0);updateMapFromToday()}else{recentreLocalMapFromStation(s)}renderStation(s)
  });
  es.addEventListener('operator',function(e){var o=JSON.parse(e.data).payload;
    D('sse','operator',o.callsign);
    updateStationField('Operator',(o.callsign||'—')+(o.name?' ('+o.name+')':''))
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
  es.addEventListener('aprs',function(e){
    var a=JSON.parse(e.data).payload;
    D('sse','aprs',a? a.length:0);
    renderAPRSOnLocalMap(a);
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
    initLocalMap(ownStationLat,ownStationLon);
    recentreLocalMap(ownStationLat,ownStationLon);
    fetchWeather(snap.station.lat,snap.station.lon);
  }
  renderStation(snap.station,snap.operator,snap.logbook,snap.rig,snap.wsjtx);
  // Active QSO
  if(snap.activeQso&&snap.activeQso.call){switchToActive();renderHero(snap.activeQso,snap.partner)}
  else{switchToOverview();renderHero(null)}
  // Stats + recent
  renderStats(snap.stats,todayQsos);renderRecentTable(snap.recent);
  // Extra info box above local map
  if(snap.solar){registerSolarModule(snap.solar)}
  if(snap.dxc&&snap.dxc.spottedBy){registerDXCModule(snap.dxc)}
  if(snap.psk&&snap.psk.total>0){registerPSKModule(snap.psk)}
  updateExtraBox();
  // APRS stations on local map
  if(snap.aprs)renderAPRSOnLocalMap(snap.aprs);
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
  $('footer-attrib').innerHTML='Map: <a href=\"https://leafletjs.com\" target=\"_blank\" rel=\"noopener\">Leaflet</a> · Tiles: <a href=\"https://www.openstreetmap.org/copyright\" target=\"_blank\" rel=\"noopener\">&copy; OpenStreetMap</a> · Solar: <a href=\"https://www.hamqsl.com/solar.html\" target=\"_blank\" rel=\"noopener\">HamQSL</a> · Spots: <a href=\"https://pskreporter.info/\" target=\"_blank\" rel=\"noopener\">PSK Reporter</a> · Weather: <a href=\"https://open-meteo.com/\" target=\"_blank\" rel=\"noopener\">Open-Meteo</a> · Callbook: <a href=\"https://www.qrz.com\" target=\"_blank\" rel=\"noopener\">QRZ.com</a>'+
    (snap.dxc&&snap.dxc.connected?' · Cluster: <span style=\"color:var(--dim)\">'+esc(snap.dxc.host||'DX Cluster')+'</span>':'');
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
  // Meta line removed — QRZ attribution moved to footer.
  $('hero-meta').textContent='';
  // Resolved reference names (SOTA/POTA/WWFF/IOTA)
  $('hero-refs').textContent=aq.refNames||'';
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
  $('hero-photo-box').style.display='';
}
function showHeroPlaceholder(call){
  $('hero-photo').style.display='none';$('hero-photo-box').style.display='none';
}
function hideHeroPhoto(){$('hero-photo').style.display='none';$('hero-placeholder').style.display='none';$('hero-photo-box').style.display='none'}

// ---- Station panel ----
function renderStation(st,op,lb,rig,wsjtx){
  if(!st)return;op=op||{};lb=lb||{};rig=rig||{};wsjtx=wsjtx||{};
  var opText=(op.callsign||'')?op.callsign+(op.name?' ('+op.name+')':''):'—';
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
  var qsosToday=st.qsosToday||0;
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
  statsFields.innerHTML=[['QSOs',qsosToday||0],['Operators',st.operators||0],['Unique calls',st.uniqueCalls||0],['DXCC',st.dxcc||0],['Grids',st.grids||0],['Bands',st.bands||0],['Modes',st.modes||0],['Longest',longestKm?Math.round(longestKm)+' km':'—'],['Rate',(st.ratePerHour||0).toFixed(1)+'/hr']].map(function(r){return'<dt>'+r[0]+'</dt><dd>'+r[1]+'</dd>'}).join('');
  renderTopQSOs();
}


// ---- Top QSOs (by distance in today's buffer) ----
function renderTopQSOs(){
  if(!todayQsos.length||ownStationLat==null){topqsosFields.innerHTML='<dt style=\"color:var(--dim)\">—</dt>';return}
  var ranked=todayQsos.map(function(q){
    return{call:q.call||'?',grid:q.grid,band:q.band||'',mode:q.mode||'',operator:q.operator||'',km:distKm(q.grid)};
  }).filter(function(r){return r.km>0}).sort(function(a,b){return b.km-a.km}).slice(0,9);
  topqsosFields.innerHTML=ranked.map(function(r,i){
    return'<dt>'+(i+1)+'.</dt><dd><strong>'+esc(r.call)+'</strong> <span style="color:var(--dim);font-size:0.78rem">'+Math.round(r.km)+' km '+esc(r.band)+' '+esc(r.mode)+(r.operator?' by '+esc(r.operator):'')+'</span></dd>';
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
  var list=qsos&&qsos.length?qsos.slice(0,8):[];
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
  while(recentBody.children.length>8)recentBody.removeChild(recentBody.lastChild);
}

// ---- Today QSO buffer ----
function appendTodayQSO(q){todayQsos.unshift(q);if(todayQsos.length>500)todayQsos.length=500}

// ---- Map (Leaflet with great-circle paths) ----
var mapCfg={drawLines:true,maxLines:150,maxMarkers:200,highlightLastQSO:true,animateActivePath:false};
var stationLayer=null,qsoMarkerLayer=null,qsoLineLayer=null,lastQsoLayer=null,activeQsoLayer=null;
var lastQso=null,activeGrid=null;
var mapLocal=null,stationLocalMarker=null,localTiles=null;

function initMap(cfg){
  if(map)return;
  if(cfg.drawLines!==undefined)mapCfg.drawLines=!!cfg.drawLines;
  if(cfg.maxLines)mapCfg.maxLines=cfg.maxLines;
  if(cfg.maxMarkers)mapCfg.maxMarkers=cfg.maxMarkers;
  if(cfg.highlightLastQSO!==undefined)mapCfg.highlightLastQSO=!!cfg.highlightLastQSO;
  if(cfg.animateActivePath!==undefined)mapCfg.animateActivePath=!!cfg.animateActivePath;
  map=L.map('map-container',{zoomControl:false,attributionControl:false}).setView([51,10],3);
  // Custom panes for layer ordering: radar below QSO paths, markers on top.
  map.createPane('cqopsRadar');map.getPane('cqopsRadar').style.zIndex=350;map.getPane('cqopsRadar').style.pointerEvents='none';
  map.createPane('cqopsGrayline');map.getPane('cqopsGrayline').style.zIndex=300;map.getPane('cqopsGrayline').style.pointerEvents='none';
  map.createPane('cqopsPath');map.getPane('cqopsPath').style.zIndex=430;map.getPane('cqopsPath').style.pointerEvents='none';
  map.createPane('cqopsActive');map.getPane('cqopsActive').style.zIndex=460;map.getPane('cqopsActive').style.pointerEvents='none';
  map.createPane('cqopsMarker');map.getPane('cqopsMarker').style.zIndex=500;
  var tiles=L.tileLayer(cfg.mapTileUrl||'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png',{maxZoom:19,attribution:cfg.mapAttrib||'&copy; OpenStreetMap'}).addTo(map);
  tiles.on('tileerror',function(e){e.tile.style.display='none'});
  // Layer groups — each on its own pane for correct z-ordering above radar.
  qsoLineLayer=L.layerGroup([],{pane:'cqopsPath'}).addTo(map);
  lastQsoLayer=L.layerGroup([],{pane:'cqopsPath'}).addTo(map);
  activeQsoLayer=L.layerGroup([],{pane:'cqopsActive'}).addTo(map);
  qsoMarkerLayer=L.layerGroup([],{pane:'cqopsMarker'}).addTo(map);
  stationLayer=L.layerGroup([],{pane:'cqopsMarker'}).addTo(map);
  // Keep Leaflet in sync with container size changes (hero toggle, resize, etc.).
  if(window.ResizeObserver){new ResizeObserver(function(){map.invalidateSize()}).observe(mapContainer)}
  // Firefox / narrow screens: the flex container may not have its final
  // computed height when Leaflet initializes. Invalidate immediately once
  // the container has height, then again after layout settles.
  (function pollMapSize(){
    if(mapContainer.clientHeight>0){
      map.invalidateSize();
      setTimeout(function(){map.invalidateSize();updateExtraBox()},150);
      return
    }
    requestAnimationFrame(pollMapSize)
  })()
  // Grayline: always-on below radar.
  enableGrayline();
  // Radar: always enabled — no toggle button.
  enableRadarLayer();
}

// ---- Local map (station-centre, ~50 km view) ----
var aprsCircleLayer=null;

function initLocalMap(lat,lon){
  if(mapLocal)return;
  var lc=document.getElementById('map-local-container');
  if(!lc)return;
  mapLocal=L.map('map-local-container',{zoomControl:false,attributionControl:false}).setView([lat,lon],11);
  mapLocal.createPane('cqopsRadar');mapLocal.getPane('cqopsRadar').style.zIndex=350;mapLocal.getPane('cqopsRadar').style.pointerEvents='none';
  mapLocal.createPane('cqopsGrayline');mapLocal.getPane('cqopsGrayline').style.zIndex=300;mapLocal.getPane('cqopsGrayline').style.pointerEvents='none';
  localTiles=L.tileLayer(mapCfg.mapTileUrl||'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png',{maxZoom:19}).addTo(mapLocal);
  localTiles.on('tileerror',function(e){e.tile.style.display='none'});
  // Station marker on local map — small, below APRS symbols.
  stationLocalMarker=L.circleMarker([lat,lon],{radius:5,color:'#007A3D',fillColor:'#007A3D',fillOpacity:0.85,weight:2.5,pane:'shadowPane'}).addTo(mapLocal);
  if(window.ResizeObserver){new ResizeObserver(function(){mapLocal.invalidateSize()}).observe(lc)}
  (function pollLocalSize(){
    if(lc.clientHeight>0){
      mapLocal.invalidateSize();
      setTimeout(function(){mapLocal.invalidateSize();updateExtraBox()},150);
      return
    }
    requestAnimationFrame(pollLocalSize)
  })()
  // Grayline on local map.
  enableGraylineLocal();
  // Add radar to local map if already enabled.
  addRadarToLocalMap();
}

function updateAprsCircle(lat,lon,radiusKm){
  if(!mapLocal)return;
  if(!aprsCircleLayer){aprsCircleLayer=L.layerGroup().addTo(mapLocal)}
  aprsCircleLayer.clearLayers();
  if(radiusKm>0&&lat!=null&&lon!=null){
    var c=L.circle([lat,lon],{
      radius:radiusKm*1000,
      color:'#007A3D',
      weight:2.5,
      opacity:0.6,
      dashArray:'10 6',
      fillColor:'#007A3D',
      fillOpacity:0.06,
      interactive:false,
      className:'aprs-range-circle'
    }).addTo(aprsCircleLayer);
    c.bringToFront();
  }
}

function recentreLocalMap(lat,lon){
  if(!mapLocal)return;
  mapLocal.setView([lat,lon],12);
  if(stationLocalMarker){stationLocalMarker.setLatLng([lat,lon])}
}

function recentreLocalMapFromStation(st){
  if(!st||!st.locator)return;
  var ll=gridToLatLon(st.locator);
  if(!ll[0])return;
  if(!mapLocal){initLocalMap(ll[0],ll[1])}
  else{recentreLocalMap(ll[0],ll[1])}
}

// ---- Extra info box above local map: module cycling ---- 
var extraModules=[];
var extraModuleIdx=0;
var extraCycleTimer=null;
var solarData=null;

function registerExtraModule(fn){extraModules.push(fn)}

function registerSolarModule(d){
  solarData=d;
  // Remove previously registered solar sub-modules.
  extraModules=extraModules.filter(function(f){return f._id!=='solar'});
  var sf=d.solarFlux||0, a=d.aIndex||0, k=d.kIndex||0, ss=d.sunspots||0;
  function sc(v,good,fair){return v<=good?'var(--success)':v<=fair?'var(--warn)':'var(--offline)'}
  function sn(v){return v||'—'}
  function cond(v,good,fair){var c=sc(v,good,fair);return'<b style="color:'+c+'">'+sn(v)+'</b>'}

  // Module 1: Solar activity (SFI + Sunspots).
  var m1=function(){
    return'<div class="extra-title">Solar Activity</div>'+
      '<div style="display:flex;flex-wrap:wrap;justify-content:center;gap:4px 12px;font-size:0.85rem">'+
      '<span><span style="color:var(--dim)">SFI</span> '+cond(sf,100,150)+'</span>'+
      '<span><span style="color:var(--dim)">Sunspots</span> <b>'+sn(ss)+'</b></span>'+
      '</div>';
  };m1._id='solar';

  // Module 2: Geomagnetic field (A + K indices).
  var m2=function(){
    return'<div class="extra-title">Geomagnetic Field</div>'+
      '<div style="display:flex;flex-wrap:wrap;justify-content:center;gap:4px 12px;font-size:0.85rem">'+
      '<span><span style="color:var(--dim)">A-index</span> '+cond(a,7,15)+'</span>'+
      '<span><span style="color:var(--dim)">K-index</span> '+cond(k,2.5,4)+'</span>'+
      '</div>';
  };m2._id='solar';

  // Module 3: Band conditions (day/night per band).
  var m3=function(){
    function bc(v){return v==='Good'?'var(--success)':v==='Fair'?'var(--warn)':'var(--offline)'}
    var html='<div class="extra-title">Band Conditions</div>'+
      '<table style="font-size:0.72rem;border-collapse:collapse;margin:0 auto;line-height:1.35">'+
      '<tr style="color:var(--dim)"><td></td><td style="padding:0 4px">Day</td><td style="padding:0 4px">Night</td></tr>';
    var bands=[['80-40','80m-40m'],['30-20','30m-20m'],['17-15','17m-15m'],['12-10','12m-10m']];
    for(var i=0;i<bands.length;i++){
      var key=bands[i][1],label=bands[i][0];
      var day=d.bandConditions? (d.bandConditions[key+'_day']||'—'):'—';
      var night=d.bandConditions? (d.bandConditions[key+'_night']||'—'):'—';
      html+='<tr>'+
        '<td style="color:var(--dim);padding-right:6px;text-align:right">'+label+'</td>'+
        '<td style="padding:0 4px;color:'+bc(day)+';font-weight:600">'+day+'</td>'+
        '<td style="padding:0 4px;color:'+bc(night)+';font-weight:600">'+night+'</td>'+
        '</tr>';
    }
    html+='</table>';return html;
  };m3._id='solar';

  extraModules.unshift(m3,m2,m1);
}

function registerDXCModule(d){
  extraModules=extraModules.filter(function(f){return f._id!=='dxc'});
  var mod=function(){
    var spotter=d.spottedBy||'?';
    var freq=d.freqKhz? (d.freqKhz/1000).toFixed(3)+' MHz' : '';
    var comment=d.comment||'';
    return'<div class="extra-title">Last Spotted By</div>'+
      '<div style="font-size:1.1rem;font-weight:700;color:var(--accent)">'+spotter+'</div>'+
      (freq?'<div style="font-size:0.78rem;color:var(--text-secondary)">'+freq+'</div>':'')+
      (comment?'<div style="font-size:0.7rem;color:var(--dim);margin-top:2px">'+comment+'</div>':'');
  };
  mod._id='dxc';
  extraModules.unshift(mod);
}

function registerPSKModule(d){
  extraModules=extraModules.filter(function(f){return f._id!=='psk'});
  var mod=function(){
    var html='<div class="extra-title">PSK Reporter</div>'+
      '<div style="font-size:0.78rem;color:var(--text-secondary);margin-bottom:3px">'+d.total+' reports</div>'+
      '<table style="font-size:0.68rem;border-collapse:collapse;margin:0 auto;line-height:1.3">';
    var order=['160m','80m','60m','40m','30m','20m','17m','15m','12m','10m','6m','4m','2m','70cm','23cm'];
    for(var i=0;i<order.length;i++){
      var b=order[i],count=d.byBand&&d.byBand[b]||0;
      if(count>0)html+='<tr><td style="color:var(--dim);padding-right:8px;text-align:right">'+b+'</td><td style="font-weight:600">'+count+'</td></tr>';
    }
    html+='</table>';return html;
  };
  mod._id='psk';
  extraModules.push(mod);
}

function cycleExtraModule(){
  if(!extraModules.length)return;
  var box=document.getElementById('map-extra-box');
  var content=document.getElementById('map-extra-content');
  if(!box||!content||!box.classList.contains('visible'))return;
  extraModuleIdx=(extraModuleIdx+1)%extraModules.length;
  content.style.opacity='0';
  setTimeout(function(){
    content.innerHTML=extraModules[extraModuleIdx]();
    content.style.opacity='1';
  },200);
}

// Show/hide the extra box and manage the cycle timer.
function updateExtraBox(){
  var box=document.getElementById('map-extra-box');
  var right=document.getElementById('map-local-right');
  if(!box||!right)return;
  if(right.clientHeight>=300){
    box.classList.add('visible');
    if(!extraModules.length){
      // Default module: CQOps marketing.
      registerExtraModule(function(){
        return '<div class="extra-title">CQOps.com</div>'+
               '<div style="font-size:0.78rem;color:var(--text-secondary)">Fast · Portable · Open Source</div>'+
               '<div style="font-size:0.7rem;color:var(--dim);margin-top:2px">Ham radio logger for the terminal</div>';
      });
    }
    // Start cycling if not already running.
    if(!extraCycleTimer){
      // Render first module immediately.
      var content=document.getElementById('map-extra-content');
      if(content)content.innerHTML=extraModules[0]();
      extraCycleTimer=setInterval(cycleExtraModule,5000);
    }
  }else{
    box.classList.remove('visible');
    if(extraCycleTimer){clearInterval(extraCycleTimer);extraCycleTimer=null}
  }
}
window.addEventListener('resize',function(){updateExtraBox();if(mapLocal)mapLocal.invalidateSize()});

// ---- RainViewer weather radar overlay ----
var radarLayer=null,radarLayerLocal=null,radarEnabled=false,radarLoading=false,radarTimer=null;
var _radarUrl='';

async function fetchJsonWithTimeout(url,ms){
  ms=ms||5000;var ctrl=new AbortController(),t=setTimeout(function(){ctrl.abort()},ms);
  try{var r=await fetch(url,{signal:ctrl.signal,cache:'no-store'});if(!r.ok)throw new Error('HTTP '+r.status);return await r.json()}
  catch(e){throw e}
  finally{clearTimeout(t)}
}

async function enableRadarLayer(){
  if(radarEnabled||radarLoading||!map)return;
  if(!navigator.onLine){return}
  radarLoading=true;
  try{
    var meta=await fetchJsonWithTimeout('https://api.rainviewer.com/public/weather-maps.json',6000);
    if(!meta||!meta.radar||!meta.radar.past||!meta.radar.past.length)throw new Error('No radar frames');
    var latest=meta.radar.past[meta.radar.past.length-1];
    if(!latest.path)throw new Error('Missing frame path');
    var url=meta.host+latest.path+'/256/{z}/{x}/{y}/2/1_1.png';
    _radarUrl=url;
    radarLayer=L.tileLayer(url,{
      pane:'cqopsRadar',opacity:0.55,maxNativeZoom:7,maxZoom:12,
      attribution:'Weather radar: <a href=\"https://www.rainviewer.com/\" target=\"_blank\" rel=\"noopener\">RainViewer</a>'
    }).addTo(map);
    radarEnabled=true;radarLoading=false;
    enableRainViewerAttribution();
    addRadarToLocalMap();
    // Refresh radar every 10 minutes.
    radarTimer=setInterval(refreshRadarLayer,600000);
  }catch(e){
    radarLoading=false;radarLayer=null;
    D('radar','enable failed',''+e);
  }
}

function disableRadarLayer(){
  if(radarLayer){map.removeLayer(radarLayer);radarLayer=null}
  if(radarLayerLocal&&mapLocal){mapLocal.removeLayer(radarLayerLocal);radarLayerLocal=null}
  radarEnabled=false;radarLoading=false;
  if(radarTimer){clearInterval(radarTimer);radarTimer=null}
}

async function refreshRadarLayer(){
  if(!radarEnabled||!map)return;
  try{
    var meta=await fetchJsonWithTimeout('https://api.rainviewer.com/public/weather-maps.json',6000);
    if(!meta||!meta.radar||!meta.radar.past||!meta.radar.past.length)return;
    var latest=meta.radar.past[meta.radar.past.length-1];
    if(!latest.path)return;
    var url=meta.host+latest.path+'/256/{z}/{x}/{y}/2/1_1.png';
    _radarUrl=url;
    var old=radarLayer;radarLayer=L.tileLayer(url,{pane:'cqopsRadar',opacity:0.55,maxNativeZoom:7,maxZoom:12}).addTo(map);
    if(old){map.removeLayer(old);old=null}
    // Refresh local map radar too.
    if(radarLayerLocal&&mapLocal){mapLocal.removeLayer(radarLayerLocal);radarLayerLocal=null}
    addRadarToLocalMap();
  }catch(e){D('radar','refresh failed',''+e)}
}

function addRadarToLocalMap(){
  if(!mapLocal||!_radarUrl)return;
  if(radarLayerLocal){mapLocal.removeLayer(radarLayerLocal);radarLayerLocal=null}
  radarLayerLocal=L.tileLayer(_radarUrl,{pane:'cqopsRadar',opacity:0.55,maxNativeZoom:7,maxZoom:12}).addTo(mapLocal);
}

// ---- Grayline / day-night terminator overlay (always-on, below radar) ----
var graylineLayer=null,graylineTimer=null,graylineLayerLocal=null,graylineTimerLocal=null;

function enableGrayline(){
  if(!map||graylineLayer||!L.terminator)return;
  graylineLayer=L.terminator({
    pane:'cqopsGrayline',
    resolution:2,
    longitudeRange:360,
    color:'#0B1220',
    fillColor:'#0B1220',
    fillOpacity:0.16,
    opacity:0.40,
    weight:1,
    interactive:false
  }).addTo(map);
  graylineTimer=window.setInterval(function(){
    if(graylineLayer)graylineLayer.setTime();
  },60000);
  D('grayline','main map enabled');
}

function enableGraylineLocal(){
  if(!mapLocal||graylineLayerLocal||!L.terminator)return;
  graylineLayerLocal=L.terminator({
    pane:'cqopsGrayline',
    resolution:2,
    longitudeRange:360,
    color:'#0B1220',
    fillColor:'#0B1220',
    fillOpacity:0.16,
    opacity:0.40,
    weight:1,
    interactive:false
  }).addTo(mapLocal);
  graylineTimerLocal=window.setInterval(function(){
    if(graylineLayerLocal)graylineLayerLocal.setTime();
  },60000);
  D('grayline','local map enabled');
}

// ---- APRS station markers on local map ----
var aprsMarkerLayer=null;

// APRS symbol sprite sheets: 16 columns × 6 rows, 24×24px per symbol.
// Grid size: 384×144 pixels.
// Sheet 0 = primary table (/), Sheet 1 = secondary table (\), Sheet 2 = overlays.
// Per APRS 1.2: alternate tables (0-9, A-Z) composite overlay chars
// from sheet 2 on top of the SECONDARY (alternate) table symbol at
// the same code position — NOT the primary table.

function _aprSpriteHTML(sheetIdx,charCode){
  if(charCode<33||charCode>126)return '';
  var idx=charCode-33,col=idx%16,row=Math.floor(idx/16);
  var url='/images/symbols/aprs-symbols-24-'+sheetIdx+'.png';
  var bgX=-(col*24),bgY=-(row*24);
  return 'background-image:url('+url+');background-repeat:no-repeat;'+
    'background-position:'+bgX+'px '+bgY+'px;background-size:384px 144px;';
}

function _aprIconHTML(sym,callsign){
  var html='<div style="text-align:center;line-height:1;position:relative;display:inline-block;">';
  var code=sym&&sym.length>1?sym.charCodeAt(1):0;
  if(!sym||sym.length<2||code<33||code>126){
    // Unrenderable symbol — orange diamond fallback.
    html+='<div style="width:24px;height:24px;display:inline-block;'+
      'background:#B45309;border:1.5px solid #fff;transform:rotate(45deg);"></div>';
  }else{
    var table=sym[0];
    if(table=='/'){
      // Primary table — simple single sprite.
      html+='<div style="width:24px;height:24px;display:inline-block;'+_aprSpriteHTML(0,code)+'"></div>';
    }else if(table=='\\'){
      // Secondary table.
      html+='<div style="width:24px;height:24px;display:inline-block;'+_aprSpriteHTML(1,code)+'"></div>';
    }else{
      // Alternate/overlay table (0-9, A-Z):
      //   base    = secondary table (sheet 1) at code position
      //   overlay = sheet 2 at table char position
      html+='<div style="position:relative;width:24px;height:24px;display:inline-block;">'+
        '<div style="position:absolute;inset:0;'+_aprSpriteHTML(1,code)+'"></div>'+
        '<div style="position:absolute;inset:0;'+_aprSpriteHTML(2,table.charCodeAt(0))+';opacity:0.9;"></div>'+
        '</div>';
    }
  }
  if(callsign)html+='<div style="font-weight:700;font-size:0.58rem;color:#fff;text-shadow:0 0 3px #000,0 0 6px #000;margin-top:1px;white-space:nowrap;">'+esc(callsign)+'</div>';
  html+='</div>';
  return html;
}

function _aprMarker(s){
  var html=_aprIconHTML(s.symbol,s.callsign);
  var icon=L.divIcon({
    className:'aprs-symbol-marker',
    iconSize:[50,36],
    iconAnchor:[25,34],
    popupAnchor:[0,-30],
    html:html
  });
  return L.marker([s.lat,s.lon],{icon:icon});
}

function renderAPRSOnLocalMap(stations){
  if(!mapLocal)return;
  if(!aprsMarkerLayer){aprsMarkerLayer=L.layerGroup().addTo(mapLocal)}
  aprsMarkerLayer.clearLayers();
  if(!stations||!stations.length)return;
  // Group by base callsign (strip trailing numeric SSID like -0, -10).
  // When both SP9SPM and SP9SPM-0 exist, only the SSID version wins.
  var groups={};
  stations.forEach(function(s){
    if(!s.callsign||!s.lat||!s.lon)return;
    var base=s.callsign;
    var dash=base.lastIndexOf('-');
    if(dash>0&&/^\d+$/.test(base.substring(dash+1)))base=base.substring(0,dash);
    if(!groups[base])groups[base]=[];
    groups[base].push(s);
  });
  var bounds=[];
  Object.values(groups).forEach(function(group){
    if(group.length===1){
      _renderAprsMarker(group[0],bounds);
    }else{
      // Prefer entries WITH a numeric SSID over bare callsigns.
      var withSSID=group.filter(function(s){
        var dash=s.callsign.lastIndexOf('-');
        return dash>0&&/^\d+$/.test(s.callsign.substring(dash+1));
      });
      if(withSSID.length>0){
        withSSID.forEach(function(s){_renderAprsMarker(s,bounds)});
      }else{
        _renderAprsMarker(group[0],bounds);
      }
    }
  });
  // Auto-fit map to show all APRS markers plus our station.
  if(bounds.length>0){
    if(ownStationLat!=null&&ownStationLon!=null)bounds.push([ownStationLat,ownStationLon]);
    mapLocal.fitBounds(bounds,{padding:[30,30],maxZoom:15});
  }
}

function _renderAprsMarker(s,bounds){
  var popup=(s.callsign||'?')+'<br>APRS';
  if(s.comment)popup+='<br>'+esc(s.comment);
  var ago=Math.round((Date.now()-new Date(s.lastHeard).getTime())/60000);
  popup+='<br>'+ago+' min ago';
  if(s.course)popup+='<br>Course: '+s.course+'°';
  if(s.speedKmh)popup+='<br>'+s.speedKmh+' km/h';
  var m=_aprMarker(s);
  m.bindPopup(popup);
  aprsMarkerLayer.addLayer(m);
  bounds.push([s.lat,s.lon]);
}

function updateMapFromToday(){
  if(!map){D('map','not initialized, skipping');return}
  map.invalidateSize();
  D('map','update',{todayQsos:todayQsos.length,hasStation:!!(ownStationLat!=null&&ownStationLon!=null)});
  // Clear all layers
  stationLayer.clearLayers();qsoMarkerLayer.clearLayers();qsoLineLayer.clearLayers();lastQsoLayer.clearLayers();activeQsoLayer.clearLayers();
  var bounds=[],hasStation=false;

  // ---- Station marker ----
  if(ownStationLat!=null&&ownStationLon!=null){
    var sm=L.circleMarker([ownStationLat,ownStationLon],{radius:9,color:'#007A3D',fillColor:'#007A3D',fillOpacity:0.85,weight:2.5});
    stationLayer.addLayer(sm);bounds.push([ownStationLat,ownStationLon]);hasStation=true;
  }

  // ---- QSO markers + lines ----
  if(todayQsos.length&&mapCfg.drawLines&&hasStation){
    var maxLines=mapCfg.maxLines||150;
    var maxMarkers=mapCfg.maxMarkers||200;
    var drawn=0,markersDrawn=0,lastQsoIdx=-1;
    // Find last QSO with coordinates
    for(var i=0;i<todayQsos.length;i++){var q=todayQsos[i];var ll=getQsoLatLon(q);if(ll){lastQsoIdx=i;break}}

    todayQsos.forEach(function(q,i){
      if(markersDrawn>=maxMarkers)return;
      var ll=getQsoLatLon(q);if(!ll)return;
      var lat=ll[0],lon=ll[1];
      // Marker
      var isLast=(i===lastQsoIdx&&mapCfg.highlightLastQSO),isActive=(activeGrid&&q.grid&&q.grid.toUpperCase()===activeGrid.toUpperCase());
      var mr=isActive?7:isLast?5.5:4;
      var mc=isActive?'#D00032':isLast?'#0080FF':'#0080FF';
      var mf=isActive?1:isLast?0.9:0.8;
      var mk=L.circleMarker([lat,lon],{radius:mr,color:mc,fillColor:mc,fillOpacity:mf,weight:isActive?3:2,opacity:0.95});
      var popup=(q.call||'')+'<br>'+(q.band||'')+' '+(q.mode||'')+'<br>'+(q.grid||'');
      if(q.timeUtc)popup+='<br>'+q.timeUtc.slice(11,16)+'Z';
      if(q.country)popup+='<br>'+q.country;
      mk.bindTooltip(q.call||'',{direction:'top'});mk.bindPopup(popup);
      qsoMarkerLayer.addLayer(mk);bounds.push([lat,lon]);
      markersDrawn++;

      // Lines — only if within limit
      if(drawn<maxLines){
        var pts=greatCirclePoints(ownStationLat,ownStationLon,lat,lon,32);
        // Include great-circle midpoint in bounds so the arc stays visible.
        bounds.push(pts[16]);
        if(isActive){
          // Active QSO: prominent dashed line with animated dash offset.
          var alOpt={color:'#D00032',weight:4,opacity:0.9,dashArray:'12 8',className:'active-path-anim'};
          activeQsoLayer.addLayer(L.polyline(pts,alOpt));
        }else if(isLast&&mapCfg.highlightLastQSO){
          // Last QSO: blue, thicker, more opaque.
          lastQsoLayer.addLayer(L.polyline(pts,{color:'#0080FF',weight:3,opacity:0.85}));
        }else{
          // Older QSO: visible above radar.
          qsoLineLayer.addLayer(L.polyline(pts,{color:'#0080FF',weight:1.8,opacity:0.50}));
        }
        drawn++;
      }
    });
  }else if(todayQsos.length&&!hasStation){
    // No station coords — still show markers without lines (capped).
    var maxMarkers=mapCfg.maxMarkers||200;
    var markersDrawn=0;
    todayQsos.forEach(function(q){
      if(markersDrawn>=maxMarkers)return;
      var ll=getQsoLatLon(q);if(!ll)return;
      var mk=L.circleMarker(ll,{radius:4,color:'#0080FF',fillColor:'#0080FF',fillOpacity:0.8,weight:2,opacity:0.95});
      mk.bindTooltip(q.call||'',{direction:'top'});
      qsoMarkerLayer.addLayer(mk);bounds.push(ll);
      markersDrawn++;
    });
  }

  // ---- Active line (drawn on top, outside today loop) ----
  if(activeGrid&&ownStationLat!==null){
    var al=gridToLatLon(activeGrid);if(al[0]){
      var actPts=greatCirclePoints(ownStationLat,ownStationLon,al[0],al[1],48);
      activeQsoLayer.addLayer(L.polyline(actPts,{color:'#D00032',weight:4,opacity:0.9,dashArray:'12 8',className:'active-path-anim'}));
      // Partner location marker — pulsing dot at the far end of the active line.
      activeQsoLayer.addLayer(L.circleMarker(al,{radius:7,color:'#D00032',fillColor:'#D00032',fillOpacity:0.35,weight:2.5,className:'partner-dot'}));
      // Include great-circle midpoint so arc stays in bounds.
      bounds.push(actPts[24]);
      bounds.push(al);
    }
  }

  // Fit bounds — but don't override active-QSO focus set by focusMapOnGrid.
  if(bounds.length>1&&!activeGrid)map.flyToBounds(bounds,{padding:[50,50],maxZoom:18});
  else if(!hasStation)map.flyTo([51,10],2);
}

function getQsoLatLon(q){if(q.lat&&q.lon)return[q.lat,q.lon];if(q.grid){var ll=gridToLatLon(q.grid);if(ll[0])return ll}return null}

function focusMapOnGrid(grid){if(!map)return;var ll=gridToLatLon(grid);if(!ll[0])return;map.invalidateSize();if(ownStationLat!=null&&ownStationLon!=null){var mid=greatCirclePoints(ownStationLat,ownStationLon,ll[0],ll[1],48)[24];var b=L.latLngBounds([[ownStationLat,ownStationLon],ll]);b.extend(mid);map.flyToBounds(b,{padding:[60,60],maxZoom:17,duration:2.5})}else{map.flyTo(ll,6,{duration:2})}}
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

// ---- Weather: Open-Meteo free API (browser-side fetch) ----
// Hidden when offline. Refreshes every 15 min or after reconnect.
var wxTimer=null,wxInterval=null,wxLat=null,wxLon=null,wxEl=null;
function weatherIcon(code,isDay){var m={
  0:isDay?'☀️':'🌙',1:'🌤️',2:'⛅',3:'☁️',45:'🌫️',48:'🌫️',
  51:'🌦️',53:'🌦️',55:'🌧️',56:'🌧️',57:'🌧️',
  61:'🌧️',63:'🌧️',65:'🌧️',66:'🌧️',67:'🌧️',
  71:'🌨️',73:'🌨️',75:'🌨️',77:'🌨️',
  80:'🌦️',81:'🌧️',82:'🌧️',
  85:'🌨️',86:'🌨️',
  95:'⛈️',96:'⛈️',99:'⛈️'
};return m[code]||(isDay?'🌡️':'🌡️')}
function wxAnimClass(code){if(code===0)return'wx-anim-sun';if(code>=1&&code<=3)return'wx-anim-cloud';if(code===45||code===48)return'wx-anim-fog';if((code>=51&&code<=57)||(code>=61&&code<=67)||(code>=80&&code<=82))return'wx-anim-rain';if((code>=71&&code<=77)||(code>=85&&code<=86))return'wx-anim-snow';if(code>=95&&code<=99)return'wx-anim-storm';return''}
function fetchWeather(lat,lon){
  if(!navigator.onLine){wxUpdateVisibility();return}
  if(wxLat===lat&&wxLon===lon)return;
  wxLat=lat;wxLon=lon;
  wxDoFetch();
  wxStartInterval();
}
function wxDoFetch(){
  if(!navigator.onLine||wxLat==null||wxLon==null)return;
  var params=new URLSearchParams({latitude:String(wxLat),longitude:String(wxLon),
    current:'temperature_2m,weather_code,wind_speed_10m,wind_gusts_10m,wind_direction_10m,precipitation,is_day',
    minutely_15:'temperature_2m,weather_code,wind_speed_10m,wind_gusts_10m,wind_direction_10m,precipitation,is_day',
    forecast_minutely_15:'13',timezone:'auto',wind_speed_unit:'kmh',precipitation_unit:'mm'});
  fetch('https://api.open-meteo.com/v1/forecast?'+params,{cache:'no-store'}).then(function(r){return r.json()}).then(function(d){
    renderWeather(d);
  }).catch(function(e){D('wx','fetch err',e.message);wxUpdateVisibility()});
}
function wxStartInterval(){
  if(wxInterval)return;
  wxInterval=setInterval(function(){
    if(navigator.onLine&&wxLat!=null&&wxLon!=null)wxDoFetch();
  },15*60*1000);
}
function wxUpdateVisibility(){
  if(!wxEl)wxEl=document.getElementById('wx-row');
  if(wxEl){
    if(navigator.onLine)wxEl.style.display='';
    else{wxEl.style.display='none';wxEl.innerHTML=''}
  }
}
// Listen for online/offline events.
window.addEventListener('online',function(){
  wxUpdateVisibility();
  if(wxLat!=null&&wxLon!=null){wxLat=null;wxLon=null;wxDoFetch();wxStartInterval()}
});
window.addEventListener('offline',function(){wxUpdateVisibility()});
function renderWeather(d){
  if(!wxEl)wxEl=document.getElementById('wx-row');
  if(!wxEl||!navigator.onLine){wxUpdateVisibility();return}
  var now=d.current||{},mn=d.minutely_15||{};
  var slots=[[0,'Now',now.temperature_2m,now.weather_code,now.wind_speed_10m,now.wind_gusts_10m,now.wind_direction_10m,now.precipitation,now.is_day]];
  var targets=[30,60,90,120,150,180];
  var nowTs=Date.now();
  for(var t=0;t<targets.length;t++){
    var ts=nowTs+targets[t]*60000,best=-1,bestD=Infinity;
    for(var i=0;i<mn.time.length;i++){var d=new Date(mn.time[i]).getTime(),diff=Math.abs(d-ts);if(diff<bestD){bestD=diff;best=i}}
    if(best>=0)slots.push([targets[t],'+'+targets[t]+'m',mn.temperature_2m[best],mn.weather_code[best],mn.wind_speed_10m[best],mn.wind_gusts_10m[best],mn.wind_direction_10m[best],mn.precipitation[best],mn.is_day[best]]);
  }
  var windArrow=function(deg){var a=['↓','↙','←','↖','↑','↗','→','↘'];return a[Math.round(deg/45)%8]||'•'};
  var html='';
  for(var s=0;s<slots.length;s++){
    var slot=slots[s],label=slot[1],temp=slot[2],code=slot[3],wSpd=slot[4],wGst=slot[5],wDir=slot[6],precip=slot[7],isDay=slot[8];
    var gustClass=wGst>50?'wx-danger':wGst>35?'wx-warn':'';
    html+='<span class="wx-slot"><span class="wx-icon '+wxAnimClass(code)+'">'+weatherIcon(code,isDay)+'</span>'+
      '<span class="wx-label">'+label+'</span>'+
      (temp!=null?'<span class="wx-temp">'+Math.round(temp)+'°</span>':'')+
      (wSpd!=null?'<span class="wx-wind '+gustClass+'">'+Math.round(wSpd)+(wGst!=null&&wGst>wSpd?'/'+Math.round(wGst):'')+'<span class="wx-wind-unit">km/h</span> '+windArrow(wDir||0)+'</span>':'')+
      (precip!=null&&precip>0?'<span class="wx-rain">'+precip.toFixed(1)+'mm</span>':'')+
      '</span>';
  }
  wxEl.className='';wxEl.innerHTML=html;wxEl.style.display='';
}

// ---- QSO sound ----
var _audioCtx=null;
function _getAudioCtx(){
  if(!_audioCtx){_audioCtx=new(window.AudioContext||window.webkitAudioContext)()}
  if(_audioCtx.state==='suspended'){_audioCtx.resume()}
  return _audioCtx;
}
// Unlock audio on first user interaction anywhere on the page.
document.addEventListener('click',function(){_getAudioCtx()},{once:true});
function playPing(){
  try{
    var ctx=_getAudioCtx();if(!ctx||ctx.state==='closed')return;
    var o=ctx.createOscillator(),g=ctx.createGain();
    o.type='sine';o.frequency.setValueAtTime(880,ctx.currentTime);
    o.frequency.exponentialRampToValueAtTime(1760,ctx.currentTime+0.08);
    g.gain.setValueAtTime(0.18,ctx.currentTime);
    g.gain.exponentialRampToValueAtTime(0.001,ctx.currentTime+0.35);
    o.connect(g);g.connect(ctx.destination);
    o.start(ctx.currentTime);o.stop(ctx.currentTime+0.35);
  }catch(e){}
}

// ---- QSO logged toast ----
var _toastTimer=null;
function showQsoToast(q){
  var t=$('qso-toast');if(!t)return;
  if(_toastTimer){clearTimeout(_toastTimer);t.className=''}
  $('qso-toast-icon').textContent='\u2713';
  $('qso-toast-call').textContent=q.call||'';
  $('qso-toast-sub').textContent=(q.band||'')+' '+(q.mode||'')+' \u2022 '+(q.rstSent||'')+'/'+(q.rstRcvd||'');
  void t.offsetWidth; // force reflow
  t.className='show';
  _toastTimer=setTimeout(function(){t.className='hide';_toastTimer=null},2200);
}

// ---- RainViewer radar attribution ----
var _rainviewerActive=false;
function enableRainViewerAttribution(){
  if(_rainviewerActive)return;_rainviewerActive=true;
  app.classList.add('has-rainviewer');
}
// Call enableRainViewerAttribution() when radar layer is added to the map.

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
