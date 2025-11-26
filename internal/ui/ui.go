package ui

import (
    "io"
    "net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    io.WriteString(w, htmlPage)
}

// Go raw string (`...`) – no JS template literals (backticks) inside.
const htmlPage = `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>SOAP Proxy Trace Viewer</title>
  <style>
    body { font-family: system-ui, sans-serif; margin: 0; display: flex; height: 100vh; }
    #list { width: 50%; border-right: 1px solid #ccc; overflow-y: auto; display: flex; flex-direction: column; }
    #detail { flex: 1; padding: 10px; overflow-y: auto; }
    table { width: 100%; border-collapse: collapse; font-size: 13px; }
    th, td { padding: 4px 6px; border-bottom: 1px solid #eee; }
    tr:hover { background: #f5f5f5; cursor: pointer; }
    pre { background: #f7f7f7; padding: 8px; white-space: pre; word-break: break-word; }
    .status-2xx { color: #2d7; }
    .status-4xx { color: #f90; }
    .status-5xx { color: #f44; }
    #filters { padding: 6px; border-bottom: 1px solid #ddd; flex-shrink: 0; }
    #filters input { margin-right: 8px; }
    #filters input[type="text"] { font-size: 12px; padding: 3px 5px; }
    #tableWrapper { flex: 1; overflow-y: auto; }
    .fail-row { background-color: #ffe6e6; }
    .fail-row:hover { background-color: #ffd6d6; }
    .fail-badge { display: inline-block; padding: 2px 6px; border-radius: 4px; background: #f44; color: #fff; font-size: 11px; margin-left: 6px; }
    .tracking-badge { display: inline-block; padding: 2px 6px; border-radius: 4px; background: #eef; color: #224; font-size: 11px; margin-left: 6px; }
    ul.related-list { padding-left: 18px; font-size: 12px; }
  </style>
</head>
<body>
  <div id="list">
    <div id="filters">
      <input type="text" id="filterPath" placeholder="Filter path...">
      <input type="text" id="filterAction" placeholder="Filter SOAPAction...">
      <input type="text" id="filterTracking" placeholder="Filter TrackingId...">
    </div>
    <div id="tableWrapper">
      <table id="traceTable">
        <thead>
          <tr>
            <th>Time</th>
            <th>SOAPAction</th>
            <th>TrackingId</th>
            <th>Status</th>
            <th>Dur (ms)</th>
          </tr>
        </thead>
        <tbody></tbody>
      </table>
    </div>
  </div>
  <div id="detail">
    <h2>Details</h2>
    <div id="detailContent">Select a request</div>
  </div>

<script>
let allTraces = [];

function escapeHtml(str) {
  if (str === null || str === undefined) return '';
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function isXmlContent(headers) {
  if (!headers) return false;
  const ct = (headers['Content-Type'] || headers['content-type']) || [];
  const val = Array.isArray(ct) ? ct[0] : ct;
  if (!val) return false;
  const v = String(val).toLowerCase();
  return v.includes('xml') || v.includes('soap');
}

function formatXml(xml) {
  if (xml === null || xml === undefined) return '';
  xml = String(xml).trim();
  if (!xml) return '';

  xml = xml.replace(/>\s*</g, '><');

  const PADDING = '  ';
  let formatted = '';
  let pad = 0;
  const tokens = xml.split(/(?=<)/g);

  for (let i = 0; i < tokens.length; i++) {
    let token = tokens[i].trim();
    if (!token) continue;

    const isClosing      = token.startsWith('</');
    const isSelfClosing  = /\/\>\s*$/.test(token);
    const isDeclaration  = token.startsWith('<?');
    const isCommentOrDtd = token.startsWith('<!--') || token.startsWith('<!DOCTYPE') || token.startsWith('<!doctype');
    const isInline       = /^<[^>]+>[^<]+<\/[^>]+>\s*$/.test(token);

    if (isClosing && !isInline) {
      pad = Math.max(pad - 1, 0);
    }

    const indent = new Array(pad + 1).join(PADDING);
    formatted += indent + token + '\n';

    if (!isClosing && !isSelfClosing && !isDeclaration && !isCommentOrDtd && !isInline && token.startsWith('<')) {
      pad++;
    }
  }

  return formatted;
}

function getHeaderValue(headers, name) {
  if (!headers) return '';
  const target = String(name).toLowerCase();
  for (const key in headers) {
    if (!Object.prototype.hasOwnProperty.call(headers, key)) continue;
    if (String(key).toLowerCase() === target) {
      const v = headers[key];
      if (Array.isArray(v)) {
        return v.length > 0 ? String(v[0]) : '';
      }
      return String(v);
    }
  }
  return '';
}

async function loadTraces() {
  const res = await fetch('/api/traces');
  const traces = await res.json();
  allTraces = traces;
  renderTraceTable();
}

function renderTraceTable() {
  const tbody = document.querySelector('#traceTable tbody');
  const pathFilter = document.getElementById('filterPath').value.toLowerCase();
  const actionFilter = document.getElementById('filterAction').value.toLowerCase();
  const trackingFilter = document.getElementById('filterTracking').value.toLowerCase();

  tbody.innerHTML = '';
  const filtered = allTraces.filter(function(t) {
    const p = (t.path || '').toLowerCase();
    const soapActionVal = (t.soapAction || getHeaderValue(t.req && t.req.headers, 'SOAPAction') || '').toLowerCase();
    const trackingId = getHeaderValue(t.req && t.req.headers, 'Trackingid').toLowerCase();
    return (!pathFilter || p.includes(pathFilter)) &&
           (!actionFilter || soapActionVal.includes(actionFilter)) &&
           (!trackingFilter || trackingId.includes(trackingFilter));
  });

  filtered.slice().reverse().forEach(function(t) {
    const tr = document.createElement('tr');
    tr.dataset.id = t.id;
    const d = new Date(t.startedAt);
    let statusClass = '';
    if (t.statusCode >= 500) statusClass = 'status-5xx';
    else if (t.statusCode >= 400) statusClass = 'status-4xx';
    else if (t.statusCode >= 200) statusClass = 'status-2xx';

    const trackingIdVal = getHeaderValue(t.req && t.req.headers, 'Trackingid');
    const soapActionVal = t.soapAction || getHeaderValue(t.req && t.req.headers, 'SOAPAction') || '';
    const failInBody = t.resp && typeof t.resp.body === 'string' &&
                       t.resp.body.toLowerCase().indexOf('fail') !== -1;

    if (failInBody) {
      tr.className = 'fail-row';
    }

    tr.innerHTML =
      '<td>' + d.toLocaleTimeString() + '</td>' +
      '<td>' + escapeHtml(soapActionVal) + '</td>' +
      '<td>' + escapeHtml(trackingIdVal || '') + '</td>' +
      '<td class="' + statusClass + '">' + (t.statusCode || '') + '</td>' +
      '<td>' + (t.durationMs || '') + '</td>';

    tr.onclick = function() { loadDetail(t.id); };
    tbody.appendChild(tr);
  });
}

async function loadDetail(id) {
  const res = await fetch('/api/traces/' + id);
  if (!res.ok) return;
  const t = await res.json();
  const el = document.getElementById('detailContent');

  const isReqXml = isXmlContent(t.req && t.req.headers);
  const isRespXml = isXmlContent(t.resp && t.resp.headers);

  const reqBodyRaw = (t.req && t.req.body) || '';
  const respBodyRaw = (t.resp && t.resp.body) || '';

  const reqBody = isReqXml ? formatXml(reqBodyRaw) : String(reqBodyRaw);
  const respBody = isRespXml ? formatXml(respBodyRaw) : String(respBodyRaw);

  const soapActionVal = t.soapAction || getHeaderValue(t.req && t.req.headers, 'SOAPAction') || '';
  const soapActionHtml = soapActionVal ? escapeHtml(soapActionVal) : '<em>(none)</em>';

  const trackingIdVal = getHeaderValue(t.req && t.req.headers, 'Trackingid');
  const trackingHtml = trackingIdVal ? escapeHtml(trackingIdVal) : '<em>(none)</em>';

  const failInBody = t.resp && typeof t.resp.body === 'string' &&
                     t.resp.body.toLowerCase().indexOf('fail') !== -1;

  // Other traces with same TrackingId
  let relatedHtml = '';
  if (trackingIdVal) {
    const related = allTraces.filter(function(x) {
      if (!x.req || !x.req.headers) return false;
      if (x.id === t.id) return false;
      return getHeaderValue(x.req.headers, 'Trackingid') === trackingIdVal;
    });

    if (related.length > 0) {
      relatedHtml += '<h4>Other requests with same TrackingId</h4><ul class="related-list">';
      related.forEach(function(x) {
        const xd = new Date(x.startedAt);
        relatedHtml += '<li>' +
          escapeHtml(xd.toLocaleTimeString()) + ' - ' +
          escapeHtml(x.method || '') + ' ' +
          escapeHtml(x.path || '') + ' (status ' + (x.statusCode || '') + ')' +
          '</li>';
      });
      relatedHtml += '</ul>';
    }
  }

  const reqHeadersJson = JSON.stringify(t.req && t.req.headers || {}, null, 2);
  const respHeadersJson = JSON.stringify(t.resp && t.resp.headers || {}, null, 2);

  let topLine = '<h3>' + escapeHtml(t.method || '') + ' ' + escapeHtml(t.path || '') + '</h3>';
  topLine += '<p><strong>SOAPAction:</strong> ' + soapActionHtml + '</p>';
  topLine += '<p><strong>TrackingId:</strong> ' + trackingHtml + '</p>';
  topLine += '<p><strong>Status:</strong> ' + (t.statusCode || '') + '</p>';
  topLine += '<p><strong>Duration:</strong> ' + (t.durationMs || '') + ' ms</p>';
  topLine += '<p><strong>Client:</strong> ' + escapeHtml(t.clientAddr || '') + '</p>';

  if (failInBody) {
    topLine += '<p><span class="fail-badge">Failure detected (body contains "Fail")</span></p>';
  } else if (trackingIdVal) {
    topLine += '<p><span class="tracking-badge">TrackingId present</span></p>';
  }

  el.innerHTML =
    topLine +
    '<h4>Request headers</h4>' +
    '<pre>' + escapeHtml(reqHeadersJson) + '</pre>' +
    '<h4>Request body</h4>' +
    '<pre>' + escapeHtml(reqBody) + '</pre>' +
    '<h4>Response headers</h4>' +
    '<pre>' + escapeHtml(respHeadersJson) + '</pre>' +
    '<h4>Response body</h4>' +
    '<pre>' + escapeHtml(respBody) + '</pre>' +
    relatedHtml;
}

document.getElementById('filterPath').addEventListener('input', renderTraceTable);
document.getElementById('filterAction').addEventListener('input', renderTraceTable);
document.getElementById('filterTracking').addEventListener('input', renderTraceTable);

loadTraces();
setInterval(loadTraces, 2000);
</script>
</body>
</html>`
