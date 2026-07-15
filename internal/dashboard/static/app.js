// CQOps Live — two-state SSE dashboard with Leaflet map
(function(){
'use strict';

// ---- Debug: enable with ?debug=1 in the URL or via TUI debug mode ----
var DEBUG=/[?&]debug=1(&|$)/.test(location.search);
function D(tag,msg,data){
  if(!DEBUG)return;
  var ts=new Date().toISOString().slice(11,23);
  if(data!==undefined){console.log('%c['+ts+'] %c'+tag+'%c '+msg,'color:#888','color:#D00032;font-weight:700','color:inherit',data)}
  else{console.log('%c['+ts+'] %c'+tag+'%c '+msg,'color:#888','color:#D00032;font-weight:700','color:inherit')}
}

var $=function(id){return document.getElementById(id)};
var app=$('app'), hdLogo=$('hd-logo'), hdLogoBox=$('hd-logo-box'), hdTitle=$('hd-title'), hdSubtitle=$('hd-subtitle');
var hdClockLocal=$('hd-clock-local'), hdClockUtc=$('hd-clock-utc'), hdSSE=$('hd-sse-status');
var heroOverview=$('hero-overview'), heroHeadline=$('hero-headline'), heroSubline=$('hero-subline'), heroStatus=$('hero-status'), heroPromo=$('hero-promo');
var heroLabel=$('hero-label'),heroCall=$('hero-call'),heroBadges=$('hero-badges'),heroIdentity=$('hero-identity'),heroMeta=$('hero-meta');
var stationFields=$('station-fields'), statsFields=$('stats-fields'), topqsosBody=$('topqsos-body');

// ---- Internet callbook link (set via display config) ----
var icbUrl=''; // URL template with {CALL} placeholder

// callLink returns an HTML anchor (or just the callsign text) based on
// whether an internet callbook provider is configured. The anchor inherits
// all styling from its parent — no color, no underline, no decoration.
function callLink(call){
  if(!icbUrl||!call)return esc(call);
  return '<a href=\"'+esc(icbUrl.replace('{CALL}',encodeURIComponent(call)))+'\" target=\"_blank\" rel=\"noopener\" style=\"color:inherit;text-decoration:none\">'+esc(call)+'</a>';
}

// ---- Data freshness timestamps ----
var freshness={wx:null,psk:null,radar:null,aprs:null};
function touchFreshness(key){freshness[key]=new Date();registerDataFreshness()}
function fmtFreshness(d){if(!d)return'—';var h=d.getHours().toString().padStart(2,'0'),m=d.getMinutes().toString().padStart(2,'0');return h+':'+m}
function registerDataFreshness(){
  var mod=function(){
    var rows=[];
    if(freshness.wx)rows.push('<span class="fr-row"><span class="fr-label">WX</span> '+fmtFreshness(freshness.wx)+'</span>');
    if(freshness.psk)rows.push('<span class="fr-row"><span class="fr-label">PSK</span> '+fmtFreshness(freshness.psk)+'</span>');
    if(freshness.radar)rows.push('<span class="fr-row"><span class="fr-label">Radar</span> '+fmtFreshness(freshness.radar)+'</span>');
    if(freshness.aprs)rows.push('<span class="fr-row"><span class="fr-label">APRS</span> '+fmtFreshness(freshness.aprs)+'</span>');
    if(!rows.length)return'';
    return'<div class="extra-title">Data Freshness</div><div class="freshness-grid">'+rows.join('')+'</div>';
  };
  mod._id='freshness';
  extraModules=extraModules.filter(function(f){return f._id!=='freshness'});
  if(freshness.wx||freshness.psk||freshness.radar||freshness.aprs)extraModules.unshift(mod);
  updateExtraBox();
}
var recentBody=$('recent-body');
var mapContainer=$('map-container');

var es=null, map=null, mainGL=null;
var ownStationLat=null,ownStationLon=null,ownStationCall='';
var todayQsos=[], displayCfg={};

// ---- Clocks ----
function updateClocks(){
  var n=new Date();
  var local=n.toLocaleTimeString([],{hour:'2-digit',minute:'2-digit',second:'2-digit'});
  var utc=n.toISOString().slice(11,19)+'Z';
  hdClockLocal.textContent='LOCAL '+local;
  hdClockUtc.textContent='UTC  '+utc;
}
updateClocks();setInterval(updateClocks,1000);

// ---- Units helpers ----
function isImperial(){return displayCfg.units==='imperial'}
function fmtDist(km){if(!km||km<=0)return'—';return isImperial()?Math.round(km*0.621371)+' mi':Math.round(km)+' km'}
function fmtTemp(c){if(c==null)return'—';return isImperial()?Math.round(c*9/5+32)+'\u00b0F':Math.round(c)+'\u00b0C'}
function fmtWind(kmh){if(kmh==null)return'—';return isImperial()?Math.round(kmh*0.621371)+' mph':Math.round(kmh)+' km/h'}
function fmtPrecip(mm){if(mm==null||mm<=0)return'';return isImperial()?(mm/25.4).toFixed(1)+' in':mm.toFixed(1)+' mm'}

// ---- State switching ----
function setState(active){
  var add=active?'mode-active':'mode-overview',rm=active?'mode-overview':'mode-active';
  if(!app.classList.contains(add)){
    app.classList.remove(rm);app.classList.add(add);
    if(map)map.invalidateSize();
    if(mapLocal)mapLocal.invalidateSize();
    // Delayed re-check — layout may take a frame to settle after hero toggles.
    setTimeout(function(){
      if(map)map.invalidateSize();
      if(mapLocal)mapLocal.invalidateSize();
    },200);
    D('state','switch → '+add);
  }
  if(active&&!todayQsos.length)updateMapFromToday()
}
function switchToOverview(){D('state','overview');setState(false);activeGrid=null;updateMapFromToday()}
function switchToActive(){D('state','active');setState(true)}

// ---- SSE ----
var sseReconnects=0;
function setSSEStatus(cls){hdSSE.className=cls;hdSSE.textContent=cls==='sse-connected'?'Online':cls==='sse-connecting'?'Connecting':'Offline'}

// ---- Disconnected overlay ----
var _discTimer=null,_discSince=null,_discTimeout=null,_discShown=false;
function scheduleDisconnected(){
  if(_discTimeout)return;
  _discSince=Date.now();
  var sub=$('disconnected-sub'),host=location.host;
  sub.textContent=host+' · '+(window._discLogbook||'');
  _discTimeout=setTimeout(function(){
    $('disconnected-overlay').classList.add('show');
    _discShown=true;
    _discTimer=setInterval(function(){
      var s=Math.floor((Date.now()-_discSince)/1000);
      var m=Math.floor(s/60),h=Math.floor(m/60),d=Math.floor(h/24);
      h=h%24;m=m%60;s=s%60;
      var txt;
      if(d>0){txt=d+'d '+String(h).padStart(2,'0')+':'+String(m).padStart(2,'0')+':'+String(s).padStart(2,'0')}
      else if(h>0){txt=String(h).padStart(2,'0')+':'+String(m).padStart(2,'0')+':'+String(s).padStart(2,'0')}
      else{txt=String(m).padStart(2,'0')+':'+String(s).padStart(2,'0')}
      $('disconnected-timer').textContent=txt;
    },1000);
  },8000);
}
function hideDisconnected(){
  if(_discTimeout){clearTimeout(_discTimeout);_discTimeout=null}
  if(_discTimer){clearInterval(_discTimer);_discTimer=null}
  _discSince=null;
  $('disconnected-overlay').classList.remove('show');
  if(_discShown){_discShown=false;setTimeout(function(){location.reload()},600)}
}

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
    if(r.frequency)updateStationField('Frequency',esc(r.frequency)+(r.band?' <span class=\"stat-badge '+bandBadgeClass(r.band)+'\">'+esc(r.band)+'</span>':'')+(r.mode?' <span class=\"stat-badge '+modeBadgeClass(r.mode)+'\">'+esc(r.mode)+'</span>':'')+(r.submode?' <span class=\"stat-badge '+modeBadgeClass(r.submode)+'\">'+esc(r.submode)+'</span>':''));
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
    var opVal=o.callsign||ownStationCall;
    updateStationField('Operator',opVal?opBadgeHTML(opVal,o.name):'—')
  });
  es.addEventListener('logbook',function(e){var lb=JSON.parse(e.data).payload;
    D('sse','logbook',lb.name);
    if(lb&&lb.name){
      document.title='CQOps - '+lb.name;window._discLogbook=lb.name;
      // Force the next today event to be accepted regardless of count.
      window._cqopsLiveLogbook=lb.name;
      // Immediately clear today QSOs so the map shows empty until the
      // fresh today event arrives — prevents stale markers from the
      // previous logbook.
      if(window._cqopsLastTodayLogbook&&window._cqopsLastTodayLogbook!==lb.name){
        todayQsos=[];updateMapFromToday();renderStats(null,[]);renderTopQSOs();
        window._cqopsLastTodayLogbook=lb.name;
      }
    }
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
    // Only guard against fewer results when we're still on the SAME
    // logbook — a fresh QSO may briefly see fewer QSOs than the
    // previous push. When the logbook changes, always replace.
    if(!t)return;
    if(window._cqopsLiveLogbook===window._cqopsLastTodayLogbook&&t.length<todayQsos.length)return;
    window._cqopsLastTodayLogbook=window._cqopsLiveLogbook;
    todayQsos=t;updateMapFromToday();renderStats(null,todayQsos);
  });
  es.addEventListener('aprs',function(e){
    var a=JSON.parse(e.data).payload;
    D('sse','aprs',a? a.length:0);
    renderAPRSOnLocalMap(a);
  });
  es.addEventListener('display',function(e){
    var d=JSON.parse(e.data).payload;
    if(!d)return;
    // Enable debug BEFORE any D() call — TUI debug mode OR URL param.
    if(d.debug&&!DEBUG){DEBUG=true;D('init','debug mode ON (from TUI) — open console (F12) for traces')}
    D('display','received',{isOnline:d.isOnline,hasCallbook:!!d.internetCallbookUrl,debug:d.debug});
    var prevOnline=window._cqopsLastOnline;
    var nowOnline=!!d.isOnline;
    if(!prevOnline&&nowOnline){
      // false→true: internet just came up — re-init with tiled Web Mercator.
      D('display','isOnline false→true — re-initializing map with tiles');
      // Refresh QR code immediately — don't wait for the next renderAll.
      _refreshQR(d);
      if(typeof map!=='undefined'&&map&&map._cqopsOfflineCRS){
        D('display','removing offline overlay, re-creating map');
        removeOfflineOverlay();
        window._mapReinitPending=true;
        // MapLibre GL scripts may not have loaded while the page was
        // offline (CDN scripts failed). Inject them dynamically before
        // creating the tiled map, otherwise initMap silently skips tiles.
        _ensureMapLibreGL(function(){
          window._mapReinitPending=false;
          initMap(Object.assign({},d,{drawLines:displayCfg.drawLines!==false,maxLines:displayCfg.maxLines||250,highlightLastQSO:displayCfg.highlightLastQSO!==false,animateActivePath:!!displayCfg.animateActivePath}));
          if(map){setTimeout(function(){map.invalidateSize()},200);setTimeout(function(){map.invalidateSize()},600);}
        });
      }else{
        D('display','map not in offline CRS mode (map='+(typeof map)+', _cqopsOfflineCRS='+(map&&map._cqopsOfflineCRS)+')');
      }
    }else if(prevOnline&&!nowOnline){
      // true→false: internet just went down — switch to offline CRS with
      // the embedded world map image so the map stays usable.
      D('display','isOnline true→false — switching to offline map');
      // Hide QR immediately — QuickChart is unreachable while offline.
      _refreshQR(d);
      if(typeof map!=='undefined'&&map&&!map._cqopsOfflineCRS){
        D('display','removing online map, re-creating with offline CRS');
        // Destroy the online map (clean up MapLibre GL / WebGL context).
        _mapPollActive=false;
        if(_mapResizeObserver){_mapResizeObserver.disconnect();_mapResizeObserver=null}
        if(map){map.remove();map=null}
        // Re-init with offline equirectangular + world map image.
        window._mapReinitPending=true;
        setTimeout(function(){
          window._mapReinitPending=false;
          initMap(Object.assign({},d,{drawLines:displayCfg.drawLines!==false,maxLines:displayCfg.maxLines||250,highlightLastQSO:displayCfg.highlightLastQSO!==false,animateActivePath:!!displayCfg.animateActivePath,isOnline:false}));
        },100);
      }
    }
    displayCfg=d;
    window._cqopsLastOnline=nowOnline;
    if(d.internetCallbookUrl){
      icbUrl=d.internetCallbookUrl;
      renderRecentTable(null);
      renderTopQSOs();
    }
  });
  es.addEventListener('heartbeat',function(){D('sse','heartbeat')});
  es.onopen=function(){sseReconnects=0;setSSEStatus('sse-connected');hideDisconnected();removeOfflineOverlay();D('sse','connected ✓')};
  es.onerror=function(){sseReconnects++;setSSEStatus('sse-disconnected');es.close();
    D('sse','error — reconnect #'+sseReconnects+' in 3s');
    scheduleDisconnected();
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
  // Enable debug from snapshot (before any display event) — covers
  // offline startup where the display event may not fire immediately.
  if(displayCfg.debug&&!DEBUG){DEBUG=true;D('init','debug mode ON (from snapshot) — open console (F12) for traces')}
  // When internet becomes available after initial offline state,
  // re-init the map with proper tile CRS instead of the fallback map.
  var nowOnline=!!(displayCfg&&displayCfg.isOnline);
  if(!window._cqopsLastOnline&&nowOnline&&typeof map!=='undefined'&&map&&map._cqopsOfflineCRS){
    D('renderAll','isOnline false→true during full render — removing offline map');
    removeOfflineOverlay();
  }
  window._cqopsLastOnline=nowOnline;
  if(displayCfg.internetCallbookUrl) icbUrl=displayCfg.internetCallbookUrl;
  // Window title — show active logbook.
  if(snap.logbook&&snap.logbook.name){document.title='CQOps - '+snap.logbook.name;window._discLogbook=snap.logbook.name;window._cqopsLiveLogbook=snap.logbook.name;window._cqopsLastTodayLogbook=snap.logbook.name}
  // Theme — apply theme class to root element.
  var wasDark=document.documentElement.classList.contains('dark');
  var theme=displayCfg.theme;
  if(!theme){try{theme=localStorage.getItem('cqops-theme')}catch(e){}}
  // Remove any previous theme class, then add the active one (except 'bright' = default).
  document.documentElement.classList.remove('dark','yl','hivis');
  if(theme&&theme!=='bright')document.documentElement.classList.add(theme);
  var isDark=document.documentElement.classList.contains('dark');
  // Persist theme so next load applies before first paint.
  try{localStorage.setItem('cqops-theme',theme||'bright')}catch(e){}
  // Update tile layers if theme changed and maps are already live.
  if(wasDark!==isDark){
    var newStyle=styleUrlForTheme(displayCfg.mapTileUrl);
    if(mainGL)mainGL.getMaplibreMap().setStyle(newStyle);
    if(localTiles)localTiles.getMaplibreMap().setStyle(styleUrlForTheme(mapCfg.mapTileUrl));
  }
  // Display config
  if(displayCfg.clubLogo){hdLogo.src=displayCfg.clubLogo;hdLogoBox.style.display=''}else{hdLogoBox.style.display='none'}
  // QR code — QuickChart API, hidden when offline.
  _refreshQR(displayCfg);
  if(displayCfg.header1){hdTitle.textContent=displayCfg.header1;heroHeadline.textContent=displayCfg.header1}
  else{hdTitle.textContent='CQOps Live';heroHeadline.textContent='CQOps Live'}
  hdSubtitle.textContent=displayCfg.header2||'Fast, portable ham radio logger';heroSubline.textContent=displayCfg.header2||'';
  // Marketing/PR line in hero when no custom header2 is configured.
  heroPromo.textContent=displayCfg.header2?'':'Powered by CQOps &mdash; cqops.com';
  // Station
  if(snap.station){
    if(snap.station.callsign)ownStationCall=snap.station.callsign;
    if(snap.station.lat&&snap.station.lon){
    ownStationLat=snap.station.lat;ownStationLon=snap.station.lon;
    initLocalMap(ownStationLat,ownStationLon);
    recentreLocalMap(ownStationLat,ownStationLon);
    updateAprsCircle(ownStationLat,ownStationLon,snap.station.aprsRadiusKm||0);
    fetchWeather(snap.station.lat,snap.station.lon);
    fetchSunTimes(snap.station.lat,snap.station.lon);
    }
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
  if(snap.aprs&&snap.aprs.length){renderAPRSOnLocalMap(snap.aprs)}
  else if(aprsMarkerLayer){aprsMarkerLayer.clearLayers()}
  // Map
  if(!window._mapReinitPending){
    D('renderAll','init map…');
    _ensureMapLibreGL(function(){
      initMap(Object.assign({},displayCfg,{drawLines:displayCfg.drawLines!==false,maxLines:displayCfg.maxLines||250,highlightLastQSO:displayCfg.highlightLastQSO!==false,animateActivePath:!!displayCfg.animateActivePath}));
    });
  } else {
    D('renderAll','map reinit pending, skipping initMap');
  }
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
    $('footer-text').innerHTML='CQOps Live v'+esc(snap.app.version)+' · <a href=\"https://cqops.com\">cqops.com</a>';
  }
  $('footer-attrib').innerHTML='Map: <a href=\"https://leafletjs.com\" target=\"_blank\" rel=\"noopener\">Leaflet</a> · Tiles: <a href=\"https://openfreemap.org\" target=\"_blank\" rel=\"noopener\">OpenFreeMap</a> &copy; <a href=\"https://www.openmaptiles.org/\" target=\"_blank\" rel=\"noopener\">OpenMapTiles</a> Data from <a href=\"https://www.openstreetmap.org/copyright\" target=\"_blank\" rel=\"noopener\">OpenStreetMap</a> · Solar: <a href=\"https://www.hamqsl.com/solar.html\" target=\"_blank\" rel=\"noopener\">HamQSL</a> · Spots: <a href=\"https://pskreporter.info/\" target=\"_blank\" rel=\"noopener\">PSK Reporter</a> · Weather: <a href=\"https://open-meteo.com/\" target=\"_blank\" rel=\"noopener\">Open-Meteo</a>'+
    (snap.dxc&&snap.dxc.connected?' · Cluster: <span style=\"color:var(--dim)\">'+esc(snap.dxc.host||'DX Cluster')+'</span>':'');
  D('renderAll','done');
}

// Track the last active QSO + flags so they survive between SSE events.
var lastActiveFlags={},lastActiveQso=null,lastPartner=null;

// ---- Integrated active panel (hero + partner merged) ----
function renderHero(aq,p){
  if(!aq||!aq.call){heroCall.textContent='';heroBadges.innerHTML='';heroIdentity.textContent='';heroMeta.textContent='';hideHeroPhoto();lastActiveFlags={};lastActiveQso=null;clearContactWeather();D('hero','cleared');return}
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
  if(aq.band)addBadge(aq.band,bandBadgeClass(aq.band));
  if(aq.mode)addBadge(aq.mode,modeBadgeClass(aq.mode));
  if(aq.submode)addBadge(aq.submode,modeBadgeClass(aq.submode));
  if(lastActiveFlags.isDupe)addBadge('DUPE','dupe');
  else if(lastActiveFlags.isNewDxcc)addBadge('NEW DXCC','success');
  else if(lastActiveFlags.isNewCall)addBadge('NEW CALL','info');
  D('hero','render',{call:aq.call,band:aq.band,mode:aq.mode,dupe:aq.isDupe,newCall:aq.isNewCall,newDxcc:aq.isNewDxcc});
  // Provider badges — clickable links to each callbook provider's callsign page.
  var provDiv = $('hero-providers');
  provDiv.innerHTML = '';
  if (p && p.callbookProviders && p.callbookProviders.length) {
    for (var i = 0; i < p.callbookProviders.length; i++) {
      var pb = p.callbookProviders[i];
      if (pb.url) {
        var a = document.createElement('a');
        a.className = 'badge provider-badge';
        a.href = pb.url;
        a.target = '_blank';
        a.rel = 'noopener';
        a.textContent = pb.name;
        provDiv.appendChild(a);
      } else {
        var s = document.createElement('span');
        s.className = 'badge provider-badge';
        s.textContent = pb.name;
        provDiv.appendChild(s);
      }
    }
  }
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
  // Contact weather — fetch when we have country info (avoid premature API calls).
  var contactGrid=aq.grid||(p&&p.grid)||'';
  var contactCountry=(p&&p.country)||(aq&&aq.country)||'';
  if(contactGrid&&contactCountry&&navigator.onLine) fetchContactWeather(contactGrid);
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

// ---- Operator badge helpers ----
function opBadgeStyle(call){
  if(!call)return'';
  var h=0,i;for(i=0;i<call.length;i++)h=(h*31+call.charCodeAt(i))&0xffff;
  var dark=document.documentElement.classList.contains('dark')||document.documentElement.classList.contains('hivis');
  if(dark)return'background:hsl('+(h%360)+',30%,18%);color:hsl('+(h%360)+',55%,82%);border:1px solid hsl('+(h%360)+',22%,28%)';
  return'background:hsl('+(h%360)+',45%,88%);color:hsl('+(h%360)+',50%,32%);border:1px solid hsl('+(h%360)+',40%,78%)';
}
function opBadgeHTML(call,name){
  if(!call)return'';
  return'<span class=\"stat-badge stat-badge-op\" style=\"'+opBadgeStyle(call)+'\">'+esc(call)+(name?' ('+esc(name)+')':'')+'</span>';
}

// ---- Station panel ----
function renderStation(st,op,lb,rig,wsjtx){
  if(!st)return;op=op||{};lb=lb||{};rig=rig||{};wsjtx=wsjtx||{};
  var opCall=op.callsign||st.callsign;
  var opHtml=opCall?opBadgeHTML(opCall,op.name):'—';
  var rigDot=rig.connected?'<span class=\"status-on\">●</span> Connected':'<span class=\"status-off\">○</span> Disconnected';
  var wsjtxDot=wsjtx.connected?'<span class=\"status-on\">●</span> Online':'<span class=\"status-off\">○</span> Offline';
  var freqHtml=rig.frequency?esc(rig.frequency)+(rig.band?' <span class=\"stat-badge '+bandBadgeClass(rig.band)+'\">'+esc(rig.band)+'</span>':'')+(rig.mode?' <span class=\"stat-badge '+modeBadgeClass(rig.mode)+'\">'+esc(rig.mode)+'</span>':'')+(rig.submode?' <span class=\"stat-badge '+modeBadgeClass(rig.submode)+'\">'+esc(rig.submode)+'</span>':''):'—';
  stationFields.innerHTML=[
    ['Operator',opHtml],['Logbook',esc(lb.name||'—')],['Locator',esc(st.locator||'—')],
    ['Radio',esc(st.radio||'—')],['Antenna',esc(st.antenna||'—')],['Power',st.powerW?st.powerW+' W':'—'],
    ['Rig',rigDot],['WSJT-X',wsjtxDot],['Frequency',freqHtml]
  ].map(function(r){return'<dt>'+r[0]+'</dt><dd id=\"sf-'+r[0]+'\">'+r[1]+'</dd>'}).join('');
}
function updateStationField(key,val){var el=document.getElementById('sf-'+key);if(el)el.innerHTML=val}

// ---- Stats ----
// Band order for frequency-consistent badge sorting.
var _bandOrder=['2190m','630m','560m','160m','80m','60m','40m','30m','20m','17m','15m','12m','10m','6m','4m','2m','1.25m','70cm','33cm','23cm','13cm','9cm','6cm','3cm'];
function _bandRank(b){var i=_bandOrder.indexOf(b);return i>=0?i:999}
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
      if(q.mode){var dm=q.submode||q.mode;modes[dm]=1}
      if(q.grid&&q.grid.length>=4)grids[q.grid.toUpperCase().substring(0,4)]=1;
    });
    qsosToday=todayBuf.length;
    st.qsosToday=qsosToday;
    st.uniqueCalls=Object.keys(calls).length;
    st.dxcc=Object.keys(dxcc).length;
    st.grids=Object.keys(grids).length;
    st.bands=Object.keys(bands).length;
    st.modes=Object.keys(modes).length;
    // Rate is always from server (QSOs in last hour) — no JS fallback.
  }
  // Longest QSO distance — scan today buffer.
  if(ownStationLat!=null&&ownStationLon!=null&&todayQsos.length){
    todayQsos.forEach(function(q){
      var d=distKm(q.grid);if(d>longestKm)longestKm=d;
    });
  }
  // Build band/mode/operator frequency maps from today QSOs for badges.
  var buf=todayBuf&&todayBuf.length?todayBuf:todayQsos;
  var bandFreq={},modeFreq={},opFreq={};
  buf.forEach(function(q){
    if(q.band)bandFreq[q.band]=(bandFreq[q.band]||0)+1;
    if(q.mode){var dm=q.submode||q.mode;modeFreq[dm]=(modeFreq[dm]||0)+1;}
    if(q.operator||ownStationCall){var o=q.operator||ownStationCall;opFreq[o]=(opFreq[o]||0)+1;}
  });
  var bandList=Object.keys(bandFreq).sort(function(a,b){return _bandRank(a)-_bandRank(b)||bandFreq[b]-bandFreq[a]});
  var modeList=Object.keys(modeFreq).sort(function(a,b){return modeFreq[b]-modeFreq[a]});
  var opList=Object.keys(opFreq).sort(function(a,b){return opFreq[b]-opFreq[a]});
  // Render fields — bands/modes as badges, operators as count+top-3.
  var rate5=st.rate5m||0,rate15=st.rate15m||0,rate60=st.rate60m||0;
  var opsTotal=st.operators||opList.length||(ownStationCall?1:0);
  statsFields.innerHTML=[
    ['QSOs',qsosToday||0],
    ['Operators','<span class="stat-op-count">'+esc(String(opsTotal))+'</span>'+opList.slice(0,3).map(function(o){return'<span class="stat-badge stat-badge-op" style="'+opBadgeStyle(o)+'">'+esc(o)+'</span>'}).join('')+(opList.length>3?'<span class="stat-op-more">…</span>':'')],
    ['Unique calls',st.uniqueCalls||0],
    ['DXCC',st.dxcc||0],
    ['Grids',st.grids||0],
    ['Bands',bandList.length?bandList.map(function(b){return'<span class="stat-badge '+bandBadgeClass(b)+'">'+esc(b)+'</span>'}).join(''):(st.bands||'—')],
    ['Modes',modeList.length?modeList.map(function(m){return'<span class="stat-badge '+modeBadgeClass(m)+'">'+esc(m)+'</span>'}).join(''):(st.modes||'—')],
    ['Longest',longestKm?fmtDist(longestKm):'—'],
    ['QSO rate','<span class=\"stat-badge rate-badge\">5m '+rate5+'</span> <span class=\"stat-badge rate-badge\">15m '+rate15+'</span> <span class=\"stat-badge rate-badge\">1h '+rate60+'</span>']
  ].map(function(r){return'<dt>'+r[0]+'</dt><dd>'+r[1]+'</dd>'}).join('');
  renderTopQSOs();
}

// ---- Session Summary (extra module above APRS map) ----
function registerSessionSummary(qsos,dxcc,grids,longestKm,rate){
  var mod=function(){
    var parts=[];
    if(qsos)parts.push('<span class="ss-item"><span class="ss-val">'+qsos+'</span> QSOs</span>');
    if(dxcc)parts.push('<span class="ss-item"><span class="ss-val">'+dxcc+'</span> DXCC</span>');
    if(grids)parts.push('<span class="ss-item"><span class="ss-val">'+grids+'</span> grids</span>');
    if(longestKm)parts.push('<span class="ss-item"><span class="ss-val">'+fmtDist(longestKm)+'</span> best</span>');
    if(rate)parts.push('<span class="ss-item"><span class="ss-val">'+rate.toFixed(1)+'/hr</span></span>');
    return'<div class="extra-title">Session</div><div class="session-summary">'+parts.join('<span class="ss-sep">|</span>')+'</div>';
  };
  mod._id='session';
  // Replace existing session module if present, otherwise add.
  extraModules=extraModules.filter(function(f){return f._id!=='session'});
  extraModules.unshift(mod);
  updateExtraBox();
}


// ---- Top QSOs (by distance in today's buffer) ----
function renderTopQSOs(){
  if(!todayQsos.length||ownStationLat==null){topqsosBody.innerHTML='<tr><td colspan=\"5\" style=\"color:var(--dim)\">—</td></tr>';return}
  var ranked=todayQsos.map(function(q){
    return{call:q.call||'?',grid:q.grid,band:q.band||'',mode:q.mode||'',submode:q.submode||'',country:q.country||'',km:distKm(q.grid)};
  }).sort(function(a,b){return b.km-a.km}).slice(0,7);
  topqsosBody.innerHTML=ranked.map(function(r,i){
    var countryText=r.country?r.country.replace(/^The\s+/,'').replace(/^Republic Of\s+/,'').replace(/^Federal Republic Of\s+/,'').trim().substring(0,22):'';
    var displayMode=r.submode||r.mode;
    var isLongest=i===0&&r.km>0;
    var rowCls=isLongest?' class=\"tq-longest\"':'';
    return'<tr'+rowCls+'>'+
      '<td>'+callLink(r.call)+'</td>'+
      '<td><span class=\"recent-badge '+bandBadgeClass(r.band)+'\">'+esc(r.band)+'</span></td>'+
      '<td><span class=\"recent-badge '+modeBadgeClass(displayMode)+'\">'+esc(displayMode)+'</span></td>'+
      '<td>'+fmtDist(r.km)+'</td>'+
      '<td><span class="recent-badge country-badge">'+esc(countryText)+'</span></td>'+
      '</tr>';
  }).join('')||'<tr><td colspan=\"5\" style=\"color:var(--dim)\">—</td></tr>';
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
  return fmtDist(km)+' '+dirs[Math.round(deg/45)%8];
}

// ---- Recent QSOs table ----
function renderRecentTable(qsos){
  var list=qsos&&qsos.length?qsos.slice(0,7):[];
  if(!list.length){recentBody.innerHTML='<tr><td colspan=\"8\" style=\"color:var(--dim)\">No QSOs yet</td></tr>';return}
  recentBody.innerHTML=list.map(function(q){
    var utc=q.timeUtc?q.timeUtc.slice(11,16).replace(':','')+'Z':'';
    var ctry=(q.country||'').replace(/^The\s+/,'').replace(/^Republic Of\s+/,'').replace(/^Federal Republic Of\s+/,'').trim().substring(0,22);
    var dist=formatDistDir(q.grid);
    var op=q.operator||ownStationCall;
    var opBadge=op?'<span class="recent-badge" style="'+opBadgeStyle(op)+'">'+esc(op)+'</span>':'';
    return'<tr><td>'+utc+'</td><td><strong>'+callLink(q.call)+'</strong></td><td>'+renderCellBadge(q.band,'band')+'</td><td>'+renderCellBadge(q.submode||q.mode,'mode')+'</td><td class="recent-op-col">'+opBadge+'</td><td>'+esc(q.rstSent||'')+'/'+esc(q.rstRcvd||'')+'</td><td title="'+esc(q.grid||'')+'">'+dist+'</td><td title="'+esc(q.country||'')+'"><span class="recent-badge country-badge">'+esc(ctry)+'</span></td></tr>';
  }).join('');
}
function renderCellBadge(val,type){
  if(!val)return'';
  var cls=type==='band'?bandBadgeClass(val):modeBadgeClass(val);
  return'<span class=\"recent-badge '+cls+'\">'+esc(val)+'</span>';
}
function prependRecentRow(q){
  var utc=q.timeUtc?q.timeUtc.slice(11,16).replace(':','')+'Z':'';
  var dist=formatDistDir(q.grid);
  var row=document.createElement('tr');row.className='new-row';
  var op=q.operator||ownStationCall;
  var opBadge=op?'<span class="recent-badge" style="'+opBadgeStyle(op)+'">'+esc(op)+'</span>':'';
  row.innerHTML='<td>'+utc+'</td><td><strong>'+callLink(q.call)+'</strong></td><td>'+renderCellBadge(q.band,'band')+'</td><td>'+renderCellBadge(q.submode||q.mode,'mode')+'</td><td class="recent-op-col">'+opBadge+'</td><td>'+esc(q.rstSent||'')+'/'+esc(q.rstRcvd||'')+'</td><td title="'+esc(q.grid||'')+'">'+dist+'</td><td><span class="recent-badge country-badge">'+esc(q.country||'')+'</span></td>';
  if(recentBody.firstChild)recentBody.insertBefore(row,recentBody.firstChild);else recentBody.appendChild(row);
  while(recentBody.children.length>7)recentBody.removeChild(recentBody.lastChild);
}

// ---- Today QSO buffer ----
function appendTodayQSO(q){todayQsos.unshift(q);if(todayQsos.length>500)todayQsos.length=500}

// ---- Map (Leaflet with great-circle paths) ----
// ---- Map style helpers (OpenFreeMap via MapLibre GL, theme-aware) ----
function suppressMissingImages(glLayer){
  var m=glLayer.getMaplibreMap();
  if(!m)return;
  m.on('styleimagemissing',function(e){
    m.addImage(e.id,{width:1,height:1,data:new Uint8ClampedArray([0,0,0,0])});
  });
}
function styleUrlForTheme(customUrl){
  if(customUrl)return customUrl;
  var root=document.documentElement.classList;
  if(root.contains('dark'))return'https://tiles.openfreemap.org/styles/fiord';
  if(root.contains('hivis'))return'https://tiles.openfreemap.org/styles/positron';
  return'https://tiles.openfreemap.org/styles/bright';
}
function qsoPathTheme(){
  var root=document.documentElement.classList;
  if(root.contains('hivis')) return{
    active:{main:'#FFD400',mainW:2.7,u:'#000000',uW:4,uOpacity:0.92,dash:null},
    last:  {main:'#FFD400',mainW:2.7,u:'#000000',uW:4,uOpacity:0.92,dash:null},
    past:  {main:'#35E6F2',mainW:2.2,u:'#000000',uW:3.2,uOpacity:0.88,dash:null},
    marker:'#72BCFF',markerActive:'#FFD400',station:'#63F7C8'
  };
  if(root.contains('dark')) return{
    active:{main:'#FF405F',mainW:2.6,u:'rgba(0,0,0,0.58)',uW:3.1},
    last:  {main:'#E7A21A',mainW:1.8,u:'rgba(0,0,0,0.58)',uW:3.1},
    past:  {main:'#E7A21A',mainW:1.8,u:'rgba(0,0,0,0.58)',uW:3.1,opacity:0.55},
    marker:'#55AEFF',markerActive:'#FF405F',station:'#50E3A4'
  };
  if(root.contains('yl')) return{
    active:{main:'#C12678',mainW:2.5,u:'rgba(255,255,255,0.74)',uW:3},
    last:  {main:'#7844B6',mainW:1.9,u:'rgba(255,255,255,0.74)',uW:3},
    past:  {main:'#7844B6',mainW:1.9,u:'rgba(255,255,255,0.74)',uW:3,opacity:0.50},
    marker:'#7650B8',markerActive:'#C12678',station:'#207A5D'
  };
  // Bright (default)
  return{
    active:{main:'#D00032',mainW:2.6,u:'rgba(255,255,255,0.74)',uW:2.8},
    last:  {main:'#0875C9',mainW:1.9,u:'rgba(255,255,255,0.74)',uW:2.8},
    past:  {main:'#0875C9',mainW:1.9,u:'rgba(255,255,255,0.74)',uW:2.8,opacity:0.50},
    marker:'#0875C9',markerActive:'#D00032',station:'#007A3D'
  };
}
function addDualPolyline(layer,pts,cfg){
  if(cfg.uW>0)layer.addLayer(L.polyline(pts,{color:cfg.u,weight:cfg.uW,opacity:cfg.uOpacity||1,lineCap:'round',lineJoin:'round'}));
  layer.addLayer(L.polyline(pts,{color:cfg.main,weight:cfg.mainW,opacity:cfg.opacity||0.9,lineCap:'round',lineJoin:'round',dashArray:cfg.dash||null,className:cfg.className||''}));
}

var mapCfg={drawLines:true,maxLines:150,maxMarkers:200,highlightLastQSO:true,animateActivePath:false};
var stationLayer=null,qsoMarkerLayer=null,qsoLineLayer=null,lastQsoLayer=null,activeQsoLayer=null;
var lastQso=null,activeGrid=null;
var mapLocal=null,stationLocalMarker=null,localTiles=null;
var _mapResizeObserver=null,_mapPollActive=false;

// _ensureMapLibreGL loads the MapLibre GL scripts dynamically if they
// weren't loaded at page start (e.g. the page opened while offline and
// the CDN <script> tags failed). Calls cb() when ready — immediately if
// L.maplibreGL is already available, or after injection + load.
// Scripts are loaded sequentially (maplibre-gl.js first, then the
// Leaflet binding) to prevent the binding from executing before the
// core library defines its global (maplibregl).
function _ensureMapLibreGL(cb){
  if(typeof L!=='undefined'&&typeof L.maplibreGL==='function'){cb();return}
  D('initMap','MapLibre GL not loaded — injecting CDN scripts');
  var s1=document.createElement('script');
  s1.src='https://unpkg.com/maplibre-gl/dist/maplibre-gl.js';
  s1.onload=function(){
    // Core library loaded — now safe to load the Leaflet binding.
    var s2=document.createElement('script');
    s2.src='https://unpkg.com/@maplibre/maplibre-gl-leaflet/leaflet-maplibre-gl.js';
    s2.onload=function(){
      D('initMap','MapLibre GL scripts loaded \u2713');
      cb();
    };
    s2.onerror=function(){
      D('initMap','MapLibre GL Leaflet binding failed — proceeding without tiles');
      cb();
    };
    document.head.appendChild(s2);
  };
  s1.onerror=function(){
    D('initMap','MapLibre GL core library failed — proceeding without tiles');
    cb();
  };
  document.head.appendChild(s1);
}

// _refreshQR loads the QR code image from QuickChart. Resets src to empty
// first to force a fresh fetch even when the browser cached a previous
// failure (e.g. the image was requested while offline). Hidden when
// offline or when no qrLink is configured.
function _refreshQR(cfg){
  var hdQR=$('hd-qr'),hdQRImg=$('hd-qr-img'),hdQRErr=$('hd-qr-err');
  if(cfg.qrLink && cfg.isOnline){
    hdQRErr.style.display='none';hdQRImg.style.display='';
    hdQRImg.onload=function(){hdQRErr.style.display='none';hdQRImg.style.display=''};
    hdQRImg.onerror=function(){hdQRErr.style.display='';hdQRImg.style.display='none'};
    hdQRImg.src='';
    hdQRImg.src='https://quickchart.io/qr?text='+encodeURIComponent(cfg.qrLink)+'&size=80&margin=1&format=svg&ecLevel=M&dark=111827&light=ffffff&_='+Date.now();
    hdQR.style.display='';
  } else {
    // Offline or no QR link configured — hide the QR panel.
    hdQR.style.display='none';
    hdQRImg.src='';
  }
}

function initMap(cfg){
  if(map)return;
  if(cfg.drawLines!==undefined)mapCfg.drawLines=!!cfg.drawLines;
  if(cfg.maxLines)mapCfg.maxLines=cfg.maxLines;
  if(cfg.maxMarkers)mapCfg.maxMarkers=cfg.maxMarkers;
  if(cfg.highlightLastQSO!==undefined)mapCfg.highlightLastQSO=!!cfg.highlightLastQSO;
  if(cfg.animateActivePath!==undefined)mapCfg.animateActivePath=!!cfg.animateActivePath;
  // When offline, use EPSG:4326 (equirectangular) to match the embedded map image.
  // When online, default Web Mercator for MapLibre tiles.
  var mapOpts={zoomControl:false,attributionControl:false};
  if(!cfg.isOnline){mapOpts.crs=L.CRS.EPSG4326;D('initMap','offline CRS — using EPSG:4326 equirectangular')}
  else{D('initMap','online — using default Web Mercator tiles')}
  map=L.map('map-container',mapOpts).setView([51,10],3);
  map._cqopsOfflineCRS=!cfg.isOnline;
  // Custom panes for layer ordering: radar below QSO paths, markers on top.
  map.createPane('cqopsRadar');map.getPane('cqopsRadar').style.zIndex=350;map.getPane('cqopsRadar').style.pointerEvents='none';
  map.createPane('cqopsGrayline');map.getPane('cqopsGrayline').style.zIndex=300;map.getPane('cqopsGrayline').style.pointerEvents='none';
  map.createPane('cqopsPath');map.getPane('cqopsPath').style.zIndex=430;map.getPane('cqopsPath').style.pointerEvents='none';
  map.createPane('cqopsActive');map.getPane('cqopsActive').style.zIndex=460;map.getPane('cqopsActive').style.pointerEvents='none';
  map.createPane('cqopsMarker');map.getPane('cqopsMarker').style.zIndex=500;
  var style=styleUrlForTheme(cfg.mapTileUrl);
  if(typeof L.maplibreGL==='function'){
    mainGL=L.maplibreGL({style:style,attributionControl:false}).addTo(map);
    suppressMissingImages(mainGL);
  }
  // Layer groups — each on its own pane for correct z-ordering above radar.
  qsoLineLayer=L.layerGroup([],{pane:'cqopsPath'}).addTo(map);
  lastQsoLayer=L.layerGroup([],{pane:'cqopsPath'}).addTo(map);
  activeQsoLayer=L.layerGroup([],{pane:'cqopsActive'}).addTo(map);
  qsoMarkerLayer=L.layerGroup([],{pane:'cqopsMarker'}).addTo(map);
  stationLayer=L.layerGroup([],{pane:'cqopsMarker'}).addTo(map);
  // Keep Leaflet in sync with container size changes (hero toggle, resize, etc.).
  if(window.ResizeObserver){
    if(_mapResizeObserver)_mapResizeObserver.disconnect();
    _mapResizeObserver=new ResizeObserver(function(){if(map)map.invalidateSize()});
    _mapResizeObserver.observe(mapContainer);
  }
  // Firefox / narrow screens: the flex container may not have its final
  // computed height when Leaflet initializes. Invalidate immediately once
  // the container has height, then again after layout settles.
  _mapPollActive=true;
  (function pollMapSize(){
    if(!_mapPollActive||!map)return;
    if(mapContainer.clientHeight>0){
      map.invalidateSize();
      setTimeout(function(){if(map)map.invalidateSize();updateExtraBox()},150);
      return
    }
    requestAnimationFrame(pollMapSize)
  })()
  // Grayline: always-on below radar.
  enableGrayline();
  // Offline map fallback — uses equirectangular CRS (EPSG:4326) matching the
  // embedded world map image. Removed when SSE reconnects and tiles load.
  if(!cfg.isOnline){
    if(!map._cqopsOffline){
      map._cqopsOffline=L.imageOverlay('/api/map-earth',[[-90,-180],[90,180]],{opacity:0.9,pane:'cqopsRadar'}).addTo(map);
    }
  }
  // Radar: always enabled — no toggle button.
  enableRadarLayer();
  // If QSO data arrived while the map was being created (cold start
  // or offline→online transition), render it now. Without this, the
  // 'today' SSE event may have called updateMapFromToday() when map
  // was null — the data was saved into todayQsos but never drawn.
  if(todayQsos&&todayQsos.length)updateMapFromToday();
}

// removeOfflineOverlay is called when SSE reconnects (internet restored).
// If the map was created with offline CRS, destroy and re-init with tiles.
function removeOfflineOverlay(){
  if(!map)return;
  if(map._cqopsOffline){map.removeLayer(map._cqopsOffline);map._cqopsOffline=null}
  if(map._cqopsOfflineCRS){
    // Disconnect observers that reference the old map instance before
    // destroying it — prevents "map is null" crashes in stale callbacks.
    _mapPollActive=false;
    if(_mapResizeObserver){_mapResizeObserver.disconnect();_mapResizeObserver=null}
    map.remove();map=null;
    // Re-init with online tiles on next renderAll.
  }
}

// ---- Local map (station-centre, ~50 km view) ----
var aprsCircleLayer=null;

function initLocalMap(lat,lon){
  if(mapLocal)return;
  var lc=document.getElementById('map-local-container');
  if(!lc)return;
  mapLocal=L.map('map-local-container',{zoomControl:false,attributionControl:false,maxZoom:18}).setView([lat,lon],11);
  mapLocal.createPane('cqopsRadar');mapLocal.getPane('cqopsRadar').style.zIndex=350;mapLocal.getPane('cqopsRadar').style.pointerEvents='none';
  mapLocal.createPane('cqopsGrayline');mapLocal.getPane('cqopsGrayline').style.zIndex=300;mapLocal.getPane('cqopsGrayline').style.pointerEvents='none';
  if(typeof L.maplibreGL==='function'){
    localTiles=L.maplibreGL({style:styleUrlForTheme(mapCfg.mapTileUrl),attributionControl:false}).addTo(mapLocal);
    suppressMissingImages(localTiles);
  }
  // Station marker on local map — small, below APRS symbols.
  stationLocalMarker=L.circleMarker([lat,lon],{radius:5,color:qsoPathTheme().station,fillColor:qsoPathTheme().station,fillOpacity:0.85,weight:2.5,pane:'shadowPane',className:'local-station-dot'}).addTo(mapLocal);
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
  // Render any APRS data that arrived before the map was ready.
  if(_pendingAPRS){
    var p=_pendingAPRS;_pendingAPRS=null;
    renderAPRSOnLocalMap(p);
  }
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
  extraModules=extraModules.filter(function(f){return f._id!=='solar' && f._id!=='band-cond'});
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

  // Module 3: Band conditions (day/night per band) — horizontal 2×2 grid.
  var m3=function(){
    function bc(v){return v==='Good'?'success':v==='Fair'?'warn':'offline'}
    var html='<div class="extra-title">Band Conditions</div>'+'<div class="band-cond-grid">';
    var bands=[['80–40','80m-40m'],['30–20','30m-20m'],['17–15','17m-15m'],['12–10','12m-10m']];
    for(var i=0;i<bands.length;i++){
      var key=bands[i][1],label=bands[i][0];
      var day=d.bandConditions? (d.bandConditions[key+'_day']||'—'):'—';
      var night=d.bandConditions? (d.bandConditions[key+'_night']||'—'):'—';
      html+='<div class="bc-block"><span class="bc-label">'+label+'</span>'+'<span class="bc-pill bc-'+bc(day)+'">D '+day+'</span>'+'<span class="bc-pill bc-'+bc(night)+'">N '+night+'</span></div>';
    }
    html+='</div>';return html;
  };m3._id='band-cond';

  extraModules.unshift(m3,m2,m1);
}

function registerDXCModule(d){
  extraModules=extraModules.filter(function(f){return f._id!=='dxc'});
  var mod=function(){
    var spotter=d.spottedBy||'?';
    var freq=d.freqKhz? (d.freqKhz/1000).toFixed(3)+' MHz' : '';
    var comment=d.comment||'';
    return'<div class="extra-title">Last Spotted By</div>'+
      '<div style="font-size:1.1rem;font-weight:700;color:var(--accent)">'+esc(spotter)+'</div>'+
      (freq?'<div style="font-size:0.78rem;color:var(--text-secondary)">'+freq+'</div>':'')+
      (comment?'<div style="font-size:0.7rem;color:var(--dim);margin-top:2px">'+esc(comment)+'</div>':'');
  };
  mod._id='dxc';
  extraModules.unshift(mod);
}

function registerPSKModule(d){
  touchFreshness('psk');
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
  // Delegate to updateExtraBox — it knows the column count and handles
  // rendering 1 or 2 modules + index advancement correctly.
  updateExtraBox();
}

// Show/hide the extra box and render modules — multiple at once on wide screens.
function updateExtraBox(){
  var box=document.getElementById('map-extra-box');
  var right=document.getElementById('map-local-right');
  if(!box||!right)return;
  if(right.clientHeight>=300){
    box.classList.add('visible');
    if(!extraModules.length){
      registerExtraModule(function(){
        return '<div class="extra-title">CQOps.com</div>'+
               '<div style="font-size:0.78rem;color:var(--text-secondary)">Fast · Portable · Open Source</div>'+
               '<div style="font-size:0.7rem;color:var(--dim);margin-top:2px">Ham radio logger for the terminal</div>';
      });
    }
    var content=document.getElementById('map-extra-content');
    if(!content)return;
    // Show 1 or 2 modules — only split into two columns when box is wide enough
    // to fit content like band conditions without cramping.
    // Band conditions always takes full width.
    var cols=Math.min(2,Math.max(1,Math.floor(box.clientWidth/240)));
    if(cols>1 && extraModules.length>1){
      var checkIdx=extraModuleIdx%extraModules.length;
      for(var ci=0;ci<cols&&ci<extraModules.length;ci++){
        var mod=extraModules[(checkIdx+ci)%extraModules.length];
        if(mod&&mod._id==='band-cond'){cols=1;break}
      }
    }
    // Always cycle through all modules.
    if(!extraCycleTimer){
      extraCycleTimer=setInterval(cycleExtraModule,5000);
    }
    // Render current slice: modules [start .. start+cols-1], wrapping around.
    var start=extraModuleIdx%extraModules.length;
    var html='<div style="display:flex;align-items:center;justify-content:center;gap:0;width:100%;height:100%">';
    for(var i=0;i<cols&&i<extraModules.length;i++){
      var idx=(start+i)%extraModules.length;
      html+='<div class="extra-module'+(i>0?' extra-module-sep':'')+'" style="flex:1;min-width:0;padding:0 10px">'+extraModules[idx]()+'</div>';
    }
    html+='</div>';
    content.innerHTML=html;
    // Always advance for next cycle.
    extraModuleIdx=(start+cols)%extraModules.length;
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
  if(!navigator.onLine||!(displayCfg&&displayCfg.isOnline)){return}
  radarLoading=true;
  try{
    var meta=await fetchJsonWithTimeout('https://api.rainviewer.com/public/weather-maps.json',6000);
    if(!meta||!meta.radar||!meta.radar.past||!meta.radar.past.length)throw new Error('No radar frames');
    var latest=meta.radar.past[meta.radar.past.length-1];
    if(!latest.path)throw new Error('Missing frame path');
    var url='/radar-proxy'+latest.path+'/256/{z}/{x}/{y}/2/1_1.png';
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
    var url='/radar-proxy'+latest.path+'/256/{z}/{x}/{y}/2/1_1.png';
    _radarUrl=url;
    var old=radarLayer;radarLayer=L.tileLayer(url,{pane:'cqopsRadar',opacity:0.55,maxNativeZoom:7,maxZoom:12}).addTo(map);
    if(old){map.removeLayer(old);old=null}
    // Refresh local map radar too.
    if(radarLayerLocal&&mapLocal){mapLocal.removeLayer(radarLayerLocal);radarLayerLocal=null}
    addRadarToLocalMap();
    touchFreshness('radar');
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
var _pendingAPRS=null; // cached stations when map not yet ready

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
  if(callsign)html+='<div style="font-weight:700;font-size:0.65rem;color:#fff;text-shadow:0 0 3px #000,0 0 6px #000;margin-top:1px;white-space:nowrap;">'+esc(callsign)+'</div>';
  html+='</div>';
  return html;
}

function _aprMarker(s){
  var html=_aprIconHTML(s.symbol,s.callsign);
  var icon=L.divIcon({
    className:'aprs-symbol-marker',
    iconSize:[50,36],
    iconAnchor:[25,12],
    popupAnchor:[0,-16],
    html:html
  });
  return L.marker([s.lat,s.lon],{icon:icon});
}

function renderAPRSOnLocalMap(stations){
  if(!mapLocal){
    // Map not yet initialised — cache the data and render when ready.
    // This handles the race between the initial /api/snapshot fetch
    // and the SSE connection on first page load.
    _pendingAPRS = stations;
    return;
  }
  touchFreshness('aprs');
  if(!aprsMarkerLayer){aprsMarkerLayer=L.layerGroup().addTo(mapLocal)}
  aprsMarkerLayer.clearLayers();
  if(!stations||!stations.length)return;
  // Group by base callsign (strip trailing numeric SSID like -0, -10).
  // When both SP9MOA and SP9MOA-0 exist, only the SSID version wins.
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
  var ago=Math.round((Date.now()-new Date(s.lastHeard).getTime())/60000);
  // Fade linearly between 30 and 60 minutes; skip entirely if older.
  var fade=1;
  if(ago>30){fade=Math.max(0,(60-ago)/30)}
  if(fade<=0)return;
  var popup=(s.callsign||'?')+'<br>APRS';
  if(s.comment)popup+='<br>'+esc(s.comment);
  popup+='<br>'+ago+' min ago';
  if(s.course)popup+='<br>Course: '+s.course+'°';
  if(s.speedKmH)popup+='<br>'+fmtWind(s.speedKmH);
  var m=_aprMarker(s);
  m.bindPopup(popup);
  if(fade<1)m.setOpacity(fade);
  aprsMarkerLayer.addLayer(m);
  bounds.push([s.lat,s.lon]);
  // Draw position trail if station has historic points.
  if(s.trail&&s.trail.length>=1){
    var pts=[];
    s.trail.forEach(function(p){
      pts.push([p.lat,p.lon]);
    });
    pts.push([s.lat,s.lon]);
    L.polyline(pts,{color:'#1565C0',weight:3,opacity:0.75*fade}).addTo(aprsMarkerLayer);
    s.trail.forEach(function(p,i){
      var frac=i/s.trail.length;
      var r=3+frac*2;
      var o=(0.45+frac*0.40)*fade;
      if(o>0.02)L.circleMarker([p.lat,p.lon],{radius:r,color:'#C62828',fillColor:'#C62828',fillOpacity:o,weight:1}).addTo(aprsMarkerLayer);
    });
  }
}

function updateMapFromToday(){
  if(!map){D('map','not initialized, skipping');return}
  map.invalidateSize();
  D('map','update',{todayQsos:todayQsos.length,hasStation:!!(ownStationLat!=null&&ownStationLon!=null)});
  // Clear all layers
  stationLayer.clearLayers();qsoMarkerLayer.clearLayers();qsoLineLayer.clearLayers();lastQsoLayer.clearLayers();activeQsoLayer.clearLayers();
  var bounds=[],hasStation=false;
  var pt=qsoPathTheme();

  // ---- Station marker ----
  if(ownStationLat!=null&&ownStationLon!=null){
    var sm=L.circleMarker([ownStationLat,ownStationLon],{radius:9,color:pt.station,fillColor:pt.station,fillOpacity:0.85,weight:2.5,className:'station-dot'});
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
      var mc=isActive?pt.markerActive:isLast?pt.marker:pt.marker;
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
        // Per-QSO starting point: use the operator's grid at QSO time
        // (myGrid) when available. Falls back to the current station
        // position for QSOs logged before this feature was added.
        var startLat=ownStationLat,startLon=ownStationLon;
        if(q.myGrid){
          var myLL=gridToLatLon(q.myGrid);
          if(myLL&&myLL[0]){startLat=myLL[0];startLon=myLL[1];}
        }
        var pts=greatCirclePoints(startLat,startLon,lat,lon,32);
        // Include great-circle midpoint in bounds so the arc stays visible.
        bounds.push(pts[16]);
        if(isActive){
          var ac=pt.active;if(ac.dash!==null){ac.dash='12 8';ac.className='active-path-anim';}
          addDualPolyline(activeQsoLayer,pts,ac);
        }else if(isLast&&mapCfg.highlightLastQSO){
          addDualPolyline(lastQsoLayer,pts,pt.last);
        }else{
          addDualPolyline(qsoLineLayer,pts,pt.past);
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
      var mk=L.circleMarker(ll,{radius:4,color:pt.marker,fillColor:pt.marker,fillOpacity:0.8,weight:2,opacity:0.95});
      mk.bindTooltip(q.call||'',{direction:'top'});
      qsoMarkerLayer.addLayer(mk);bounds.push(ll);
      markersDrawn++;
    });
  }

  // ---- Active line (drawn on top, outside today loop) ----
  if(activeGrid&&ownStationLat!==null){
    var al=gridToLatLon(activeGrid);if(al[0]){
      var actPts=greatCirclePoints(ownStationLat,ownStationLon,al[0],al[1],48);
      var ac=pt.active;ac.dash='12 8';ac.className='active-path-anim';
      addDualPolyline(activeQsoLayer,actPts,ac);
      // Partner location marker — pulsing dot at the far end of the active line.
      activeQsoLayer.addLayer(L.circleMarker(al,{radius:7,color:pt.active.main,fillColor:pt.active.main,fillOpacity:0.35,weight:2.5,className:'partner-dot'}));
      // Include great-circle midpoint so arc stays in bounds.
      bounds.push(actPts[24]);
      bounds.push(al);
    }
  }

  // Fit bounds — but don't override active-QSO focus set by focusMapOnGrid.
  if(bounds.length>1&&!activeGrid)map.flyToBounds(bounds,{padding:[50,50],maxZoom:18});
  else if(hasStation&&ownStationLat!=null)map.flyTo([ownStationLat,ownStationLon],6);
  else if(!hasStation)map.flyTo([51,10],2);
}

function getQsoLatLon(q){if(q.lat&&q.lon)return[q.lat,q.lon];if(q.grid){var g=q.grid.toUpperCase();if(g==='AA00AA')return null;var ll=gridToLatLon(q.grid);if(ll[0])return ll}return null}

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
  var c0=grid.charCodeAt(0),c1=grid.charCodeAt(1);
  if(c0<65||c0>82||c1<65||c1>82)return[0,0]; // field must be A-R
  var lon=(c0-65)*20-180,lat=(c1-65)*10-90;
  lon+=(grid.charCodeAt(2)-48)*2;lat+=(grid.charCodeAt(3)-48)*1;
  if(grid.length>=6){
    var c4=grid.charCodeAt(4),c5=grid.charCodeAt(5);
    if(c4<65||c4>88||c5<65||c5>88)return[0,0]; // subsquare must be A-X
    lon+=(c4-65)*(5/60);
    lat+=(c5-65)*(2.5/60);
    if(grid.length>=8){
      lon+=(grid.charCodeAt(6)-48)*(0.5/60);
      lat+=(grid.charCodeAt(7)-48)*(0.25/60);
      if(grid.length>=10){
        var c8=grid.charCodeAt(8),c9=grid.charCodeAt(9);
        if(c8<65||c8>88||c9<65||c9>88)return[0,0];
        lon+=(c8-65)*(0.5/60/24);
        lat+=(c9-65)*(0.25/60/24);
        lon+=0.5/60/48;lat+=0.25/60/48;        // centre of 10-char
      }else{
        lon+=0.25/60;lat+=0.125/60;             // centre of 8-char
      }
    }else{
      lon+=2.5/60;lat+=1.25/60;                 // centre of 6-char
    }
  }else{
    lon+=1;lat+=0.5;                             // centre of 4-char
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
// ---- Centralized band / mode styling ----
var _modeGroups={cw:1,cwr:1,phone:1,ssb:1,usb:1,lsb:1,am:1,fm:1,nfm:1,digi:1,ft8:1,ft4:1,rtty:1,psk31:1,psk63:1,jt65:1,jt9:1,mfsk:1,olivia:1,hell:1,fsk441:1,jt6m:1,pkt:1,data:1,mfsk16:1,domino:1,thor:1,ros:1,js8:1,fst4:1,fst4w:1,q65:1,msk144:1,sstv:1,fax:1};
function bandBadgeClass(band){return'band-badge band-'+(band||'').toLowerCase().replace(/[^a-z0-9]/g,'')}
function modeBadgeClass(mode){
  var m=(mode||'').toLowerCase().trim();
  if(_modeGroups[m]===1){if(m==='cw'||m==='cwr')return'mode-badge mode-cw';if(m==='ssb'||m==='usb'||m==='lsb'||m==='am'||m==='fm'||m==='nfm'||m==='phone')return'mode-badge mode-phone';return'mode-badge mode-digi'}
  return'mode-badge mode-other';
}

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
    hourly:'temperature_2m,weather_code,wind_speed_10m,wind_gusts_10m,wind_direction_10m,precipitation,is_day',
    forecast_minutely_15:'13',forecast_hours:'25',timezone:'auto',wind_speed_unit:'kmh',precipitation_unit:'mm'});
  fetch('https://api.open-meteo.com/v1/forecast?'+params,{cache:'no-store'}).then(function(r){return r.json()}).then(function(d){
    renderWeather(d);
  }).catch(function(e){D('wx','fetch err',e.message);wxUpdateVisibility()});
}
function wxStartInterval(){
  if(wxInterval)return;
  wxInterval=setInterval(function(){
    if(navigator.onLine&&displayCfg&&displayCfg.isOnline&&wxLat!=null&&wxLon!=null)wxDoFetch();
  },15*60*1000);
}
function wxUpdateVisibility(){
  if(!wxEl)wxEl=document.getElementById('wx-row');
  if(wxEl){
    if(navigator.onLine&&displayCfg&&displayCfg.isOnline)wxEl.style.display='';
    else{wxEl.style.display='none';wxEl.innerHTML=''}
  }
  if(map)map.invalidateSize();if(mapLocal)mapLocal.invalidateSize();
}
// Listen for online/offline events.
window.addEventListener('online',function(){
  if(!(displayCfg&&displayCfg.isOnline))return;
  wxUpdateVisibility();
  if(wxLat!=null&&wxLon!=null){wxLat=null;wxLon=null;wxDoFetch();wxStartInterval()}
});
window.addEventListener('offline',function(){wxUpdateVisibility();clearContactWeather()});
function renderWeather(d){
  if(!wxEl)wxEl=document.getElementById('wx-row');
  if(!wxEl||!navigator.onLine){wxUpdateVisibility();return}
  var now=d.current||{},mn=d.minutely_15||{},hr=d.hourly||{};
  var slots=[[0,'Now',now.temperature_2m,now.weather_code,now.wind_speed_10m,now.wind_gusts_10m,now.wind_direction_10m,now.precipitation,now.is_day]];
  // Short-range (15-min intervals from minutely_15 API).
  var shortTargets=[30,60]; // +0.5h, +1h
  var nowTs=Date.now();
  for(var t=0;t<shortTargets.length;t++){
    var ts=nowTs+shortTargets[t]*60000,best=-1,bestD=Infinity;
    for(var i=0;i<mn.time.length;i++){var rowTime=new Date(mn.time[i]).getTime(),diff=Math.abs(rowTime-ts);if(diff<bestD){bestD=diff;best=i}}
    if(best>=0){var h=shortTargets[t]/60;slots.push([shortTargets[t],'+'+h+'h',mn.temperature_2m[best],mn.weather_code[best],mn.wind_speed_10m[best],mn.wind_gusts_10m[best],mn.wind_direction_10m[best],mn.precipitation[best],mn.is_day[best]]);}
  }
  // Longer-range (hourly API).
  var longTargets=[2,3,5,8,12,24]; // +2h, +3h, +5h, +8h, +12h, +24h
  for(var t=0;t<longTargets.length;t++){
    var ts=nowTs+longTargets[t]*3600000,best=-1,bestD=Infinity;
    for(var i=0;i<hr.time.length;i++){var rowTime=new Date(hr.time[i]).getTime(),diff=Math.abs(rowTime-ts);if(diff<bestD){bestD=diff;best=i}}
    if(best>=0){slots.push([longTargets[t]*60,'+'+longTargets[t]+'h',hr.temperature_2m[best],hr.weather_code[best],hr.wind_speed_10m[best],hr.wind_gusts_10m[best],hr.wind_direction_10m[best],hr.precipitation[best],hr.is_day[best]]);}
  }
  var windArrow=function(deg){var a=['↓','↙','←','↖','↑','↗','→','↘'];return a[Math.round(deg/45)%8]||'•'};
  var html='';
  for(var s=0;s<slots.length;s++){
    var slot=slots[s],label=slot[1],temp=slot[2],code=slot[3],wSpd=slot[4],wGst=slot[5],wDir=slot[6],precip=slot[7],isDay=slot[8];
    html+='<span class="wx-slot"><span class="wx-icon '+wxAnimClass(code)+'">'+weatherIcon(code,isDay)+'</span>'+
      '<span class="wx-label">'+label+'</span>'+
      (temp!=null?'<span class="wx-temp">'+fmtTemp(temp)+'</span>':'')+
      (wSpd!=null?'<span class="wx-wind">'+fmtWind(wSpd)+' <span class="wx-wind-dir">'+windArrow(wDir||0)+'</span></span>':'')+
      (precip!=null&&precip>0?'<span class="wx-rain">'+fmtPrecip(precip)+'</span>':'')+
      '</span>';
  }
  wxEl.className='';wxEl.innerHTML=html;wxEl.style.display='';
  touchFreshness('wx');
  // Weather row changed height — maps need recalculation.
  if(map)map.invalidateSize();if(mapLocal)mapLocal.invalidateSize();
}

// ---- Contact weather (hero box) — fetched per-contact when country+grid known ----
var _contactWxGrid=null;
function fetchContactWeather(grid){
  if(!grid||!navigator.onLine)return;
  if(_contactWxGrid===grid)return;
  _contactWxGrid=grid;
  var ll=gridToLatLon(grid);if(!ll[0])return;
  var params=new URLSearchParams({latitude:String(ll[0]),longitude:String(ll[1]),
    current:'temperature_2m,weather_code,wind_speed_10m,wind_direction_10m,is_day',
    timezone:'auto',wind_speed_unit:'kmh'});
  fetch('https://api.open-meteo.com/v1/forecast?'+params,{cache:'no-store'}).then(function(r){return r.json()}).then(function(d){
    renderContactWeather(d);
  }).catch(function(e){D('wx','contact fetch err',e.message)});
}
function renderContactWeather(d){
  var hwb=document.getElementById('hero-weather-box');if(!hwb||!navigator.onLine)return;
  var c=d.current||{},windArrow=function(deg){var a=['↓','↙','←','↖','↑','↗','→','↘'];return a[Math.round(deg/45)%8]||'•'};
  document.getElementById('hero-wx-icon').innerHTML='<span class="'+wxAnimClass(c.weather_code||0)+'">'+weatherIcon(c.weather_code||0,c.is_day)+'</span>';
  document.getElementById('hero-wx-temp').textContent=fmtTemp(c.temperature_2m);
  document.getElementById('hero-wx-wind').textContent=c.wind_speed_10m!=null?fmtWind(c.wind_speed_10m)+' '+windArrow(c.wind_direction_10m||0):'';
  hwb.classList.add('visible');
  if(map)map.invalidateSize();if(mapLocal)mapLocal.invalidateSize();
}
function clearContactWeather(){
  _contactWxGrid=null;
  var hwb=document.getElementById('hero-weather-box');if(hwb)hwb.classList.remove('visible');
  if(map)map.invalidateSize();if(mapLocal)mapLocal.invalidateSize();
}

// ---- Sunrise / sunset (daily, cached per lat/lon/date) ----
var _sunCache='',_sunEl=null;
function fetchSunTimes(lat,lon){
  if(!navigator.onLine||lat==null||lon==null)return;
  var today=new Date().toISOString().slice(0,10);
  var key=lat.toFixed(2)+','+lon.toFixed(2)+','+today;
  if(_sunCache===key)return;
  _sunCache=key;
  var params=new URLSearchParams({
    latitude:String(lat),longitude:String(lon),
    daily:'sunrise,sunset',timezone:'auto',forecast_days:'1'
  });
  fetch('https://api.open-meteo.com/v1/forecast?'+params,{cache:'no-store'}).then(function(r){return r.json()}).then(function(d){
    renderSunTimes(d);
  }).catch(function(e){D('sun','fetch err',e.message)});
}
function renderSunTimes(d){
  if(!_sunEl)_sunEl=document.getElementById('hd-sun');
  if(!_sunEl||!navigator.onLine)return;
  var sun=d.daily||{};
  var rise=sun.sunrise?sun.sunrise[0]:'',set=sun.sunset?sun.sunset[0]:'';
  if(!rise||!set){_sunEl.innerHTML='';return}
  var rt=rise.slice(11,16),st=set.slice(11,16);
  _sunEl.innerHTML='<span class="sun-icon">\u2609\u2191</span>&nbsp;'+rt+'&nbsp;&nbsp;<span class="sun-icon">\u2609\u2193</span>&nbsp;'+st;
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

})();
