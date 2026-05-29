(function () {
  function escHtml(s) {
    return String(s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  function showToast(title, description, color, duration) {
    var div = document.createElement('div');
    div.setAttribute('data-tui-toast', '');
    div.setAttribute('data-tui-toast-duration', String(duration));
    div.style.cssText = 'position:fixed;top:1rem;right:1rem;z-index:9999;width:100%;max-width:420px;pointer-events:auto';

    var card = document.createElement('div');
    card.style.cssText = 'position:relative;background:var(--popover,#fff);color:var(--popover-foreground,#0f172a);border-radius:.5rem;box-shadow:0 4px 6px -1px rgb(0 0 0/.1),0 2px 4px -2px rgb(0 0 0/.1);border:1px solid var(--border,#e2e8f0);padding:1rem 2.5rem 1rem 1rem;display:flex;align-items:center;gap:.75rem;overflow:hidden';

    var iconNS = 'http://www.w3.org/2000/svg';
    var svg = document.createElementNS(iconNS, 'svg');
    svg.setAttribute('width', '22');
    svg.setAttribute('height', '22');
    svg.setAttribute('viewBox', '0 0 24 24');
    svg.setAttribute('fill', 'none');
    svg.setAttribute('stroke', 'currentColor');
    svg.setAttribute('stroke-width', '2');
    svg.setAttribute('stroke-linecap', 'round');
    svg.setAttribute('stroke-linejoin', 'round');
    svg.style.cssText = 'color:' + color + ';flex-shrink:0';
    var circle = document.createElementNS(iconNS, 'circle');
    circle.setAttribute('cx', '12'); circle.setAttribute('cy', '12'); circle.setAttribute('r', '10');
    var path = document.createElementNS(iconNS, 'path');
    path.setAttribute('d', 'm9 12 2 2 4-4');
    svg.appendChild(circle);
    svg.appendChild(path);
    card.appendChild(svg);

    var span = document.createElement('span');
    span.style.cssText = 'flex:1;min-width:0';
    if (title) {
      var p1 = document.createElement('p');
      p1.style.cssText = 'font-size:.875rem;font-weight:600;overflow:hidden;text-overflow:ellipsis;white-space:nowrap';
      p1.textContent = title;
      span.appendChild(p1);
    }
    if (description) {
      var p2 = document.createElement('p');
      p2.style.cssText = 'font-size:.875rem;opacity:.9;margin-top:.25rem';
      p2.textContent = description;
      span.appendChild(p2);
    }
    card.appendChild(span);

    var btn = document.createElement('button');
    btn.setAttribute('data-tui-toast-dismiss', '');
    btn.setAttribute('type', 'button');
    btn.style.cssText = 'position:absolute;top:.5rem;right:.5rem;opacity:.6;cursor:pointer;background:none;border:none;padding:.25rem;border-radius:.25rem;display:flex;align-items:center;line-height:1';
    btn.onmouseover = function () { this.style.opacity = 1; };
    btn.onmouseout = function () { this.style.opacity = 0.6; };
    var xSvg = document.createElementNS(iconNS, 'svg');
    xSvg.setAttribute('width', '16'); xSvg.setAttribute('height', '16');
    xSvg.setAttribute('viewBox', '0 0 24 24'); xSvg.setAttribute('fill', 'none');
    xSvg.setAttribute('stroke', 'currentColor'); xSvg.setAttribute('stroke-width', '2');
    xSvg.setAttribute('stroke-linecap', 'round'); xSvg.setAttribute('stroke-linejoin', 'round');
    var xPath1 = document.createElementNS(iconNS, 'path'); xPath1.setAttribute('d', 'M18 6 6 18');
    var xPath2 = document.createElementNS(iconNS, 'path'); xPath2.setAttribute('d', 'm6 6 12 12');
    xSvg.appendChild(xPath1); xSvg.appendChild(xPath2);
    btn.appendChild(xSvg);
    card.appendChild(btn);

    div.appendChild(card);
    document.body.appendChild(div);
  }

  function showErrorToast(title, description) {
    showToast(title, description, '#ef4444', 0);
  }

  function showSuccessToast(title, description) {
    showToast(title, description, '#22c55e', 5000);
  }

  document.addEventListener('vkb:toast', function (evt) {
    var d = evt.detail || {};
    if (d.variant === 'error') showErrorToast(d.title || 'Error', d.message || '');
    else if (d.variant === 'success') showSuccessToast(d.title || 'Success', d.message || '');
    else showToast(d.title || 'Warning', d.message || '', '#f59e0b', 5000);
  });

  document.addEventListener('htmx:responseError', function (evt) {
    var xhr = evt.detail.xhr;
    var status = xhr.status;
    var msg = '';
    try { var json = JSON.parse(xhr.responseText); msg = json.message || json.error || ''; } catch (_) { }
    showErrorToast('Error ' + status, msg);
  });

  document.addEventListener('htmx:sendError', function () {
    showErrorToast('Network Error', 'Could not reach the server.');
  });
})();