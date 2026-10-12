(function () {
  "use strict";
  var $ = function (sel, root) { return (root || document).querySelector(sel); };
  var $$ = function (sel, root) { return Array.prototype.slice.call((root || document).querySelectorAll(sel)); };

  // Theme: auto (system) -> light -> dark. Storage may be unavailable.
  var root = document.documentElement;
  var themeBtn = $("#theme");
  function setTheme(t) {
    if (t === "light" || t === "dark") { root.setAttribute("data-theme", t); } else { root.removeAttribute("data-theme"); }
    if (themeBtn) { themeBtn.textContent = "Theme: " + (t || "auto"); }
    try { localStorage.setItem("tfplanlens-theme", t || ""); } catch (e) { /* ignore */ }
  }
  var saved = "";
  try { saved = localStorage.getItem("tfplanlens-theme") || ""; } catch (e) { /* ignore */ }
  setTheme(saved);
  if (themeBtn) {
    themeBtn.addEventListener("click", function () {
      var cur = root.getAttribute("data-theme") || "";
      setTheme(cur === "" ? "light" : cur === "light" ? "dark" : "");
    });
  }

  // Danger side panel: open by default, each resource collapsed under its name.
  var danger = $("#danger"), dToggle = $("#danger-toggle"), dClose = $("#danger-close"), dAll = $("#danger-all");
  if (danger && dToggle && dClose && dAll) {
    var showDanger = function (on) {
      danger.hidden = !on;
      document.body.classList.toggle("danger-open", on);
      dToggle.setAttribute("aria-expanded", on ? "true" : "false");
    };
    var setAll = function (open) {
      $$("details.finding", danger).forEach(function (d) { d.open = open; });
      dAll.textContent = open ? "Collapse all" : "Expand all";
    };
    dToggle.addEventListener("click", function () { showDanger(danger.hidden); });
    dClose.addEventListener("click", function () { showDanger(false); dToggle.focus(); });
    dAll.addEventListener("click", function () {
      setAll($$("details.finding", danger).some(function (d) { return !d.open; }));
    });
    window.addEventListener("beforeprint", function () { setAll(true); });
  }

  // Filter, search and group the changes table.
  var tbody = $("#rows");
  if (!tbody) { return; }
  var rows = $$("tr.row", tbody);
  var q = $("#q"), act = $("#action"), grp = $("#group"), hideQuiet = $("#hidequiet"), count = $("#count");

  function render() {
    var term = (q.value || "").toLowerCase().trim();
    var a = act.value, g = grp.value;
    $$("tr.group", tbody).forEach(function (n) { n.parentNode.removeChild(n); });
    var shown = rows.filter(function (r) {
      var ok = (!term || r.getAttribute("data-text").indexOf(term) !== -1) &&
               (!a || r.getAttribute("data-action") === a) &&
               !(hideQuiet.checked && (r.getAttribute("data-action") === "no-op" || r.getAttribute("data-action") === "read") && r.getAttribute("data-quiet") === "1");
      r.classList.toggle("hidden", !ok);
      return ok;
    });
    var ordered = rows.slice().sort(function (x, y) {
      if (g) {
        var kx = x.getAttribute("data-" + g), ky = y.getAttribute("data-" + g);
        if (kx !== ky) { return kx < ky ? -1 : 1; }
      }
      return Number(x.getAttribute("data-idx")) - Number(y.getAttribute("data-idx"));
    });
    var last = null;
    ordered.forEach(function (r) {
      if (g && !r.classList.contains("hidden")) {
        var key = r.getAttribute("data-" + g);
        if (key !== last) {
          var n = shown.filter(function (s) { return s.getAttribute("data-" + g) === key; }).length;
          var tr = document.createElement("tr");
          tr.className = "group";
          var td = document.createElement("td");
          td.colSpan = 5;
          td.textContent = key + " (" + n + ")";
          tr.appendChild(td);
          tbody.appendChild(tr);
          last = key;
        }
      }
      tbody.appendChild(r);
    });
    count.textContent = "Showing " + shown.length + " of " + rows.length;
  }
  [q, act, grp, hideQuiet].forEach(function (el) { el.addEventListener("input", render); el.addEventListener("change", render); });
  render();

  // Detail drawer.
  var drawer = $("#drawer"), body = $("#drawer-body"), lastFocus = null;
  function open(row) {
    var tpl = document.getElementById("d" + row.getAttribute("data-idx"));
    if (!tpl) { return; }
    while (body.firstChild) { body.removeChild(body.firstChild); }
    body.appendChild(tpl.content.cloneNode(true));
    drawer.hidden = false;
    lastFocus = row;
    $("#drawer-close").focus();
  }
  function close() { drawer.hidden = true; if (lastFocus) { lastFocus.focus(); } }
  tbody.addEventListener("click", function (e) {
    var t = e.target;
    while (t && t !== tbody && !(t.classList && t.classList.contains("row"))) { t = t.parentNode; }
    if (t && t !== tbody) { open(t); }
  });
  tbody.addEventListener("keydown", function (e) {
    if ((e.key === "Enter" || e.key === " ") && e.target.classList.contains("row")) { e.preventDefault(); open(e.target); }
  });
  $("#drawer-close").addEventListener("click", close);
  document.addEventListener("keydown", function (e) { if (e.key === "Escape" && !drawer.hidden) { close(); } });
})();
