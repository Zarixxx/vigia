const body = document.body;
const rowsContainer = document.querySelector("#certificate-rows");
const searchInput = document.querySelector("#search-input");
const filterButtons = [...document.querySelectorAll(".filter-tab")];
const dialog = document.querySelector("#target-dialog");
let activeFilter = "all";
let activeView = "overview";
let localAPI = false;
let liveCertificates = [];

const pageCopy = {
  overview: {
    breadcrumb: "Resumen",
    title: "Tu perímetro,<br><em>bajo control.</em>",
    description: "Una lectura clara de los certificados que protegen tus servicios.",
    inventory: "Inventario de certificados",
  },
  certificates: {
    breadcrumb: "Certificados",
    title: "Cada certificado,<br><em>en su sitio.</em>",
    description: "Explora los objetivos de muestra y filtra por estado o fecha de vencimiento.",
    inventory: "Todos los certificados",
  },
  findings: {
    breadcrumb: "Hallazgos",
    title: "Lo importante,<br><em>primero.</em>",
    description: "Hallazgos abiertos de muestra que requieren revisión.",
    inventory: "Hallazgos abiertos",
  },
};

const statusNames = {
  critical: "Crítico",
  warning: "Aviso",
  healthy: "Correcto",
  error: "Sin conexión",
};

function displayTarget(item) {
  const host = item.host.includes(":") ? `[${item.host}]` : item.host;
  return `${host}:${item.port}`;
}

function formatDate(value) {
  if (!value) return "—";
  return new Date(value).toLocaleDateString("es-ES", { day: "2-digit", month: "short", year: "numeric" });
}

function makeCell(text, className = "") {
  const cell = document.createElement("td");
  cell.textContent = text || "—";
  if (className) cell.className = className;
  return cell;
}

function createCertificateRow(item) {
  const row = document.createElement("tr");
  const status = statusNames[item.status] ? item.status : "error";
  const target = displayTarget(item);
  row.dataset.status = status;
  row.dataset.days = String(item.days_left);
  row.append(makeCell("", "target-cell"));
  const targetCell = row.firstElementChild;
  const targetName = document.createElement("span");
  targetName.className = "target-name";
  targetName.textContent = target;
  const targetMeta = document.createElement("span");
  targetMeta.className = "target-meta";
  targetMeta.textContent = item.error || `Sondeado ${new Date(item.scanned_at).toLocaleTimeString("es-ES", { hour: "2-digit", minute: "2-digit" })}`;
  targetCell.replaceChildren(targetName, targetMeta);

  row.append(makeCell(item.issuer));
  row.append(makeCell(item.error ? "—" : formatDate(item.not_after)));
  row.append(makeCell(item.error ? "—" : String(item.days_left).padStart(2, "0"), `days-cell${status === "critical" ? " days-critical" : status === "warning" ? " days-warning" : ""}`));
  const statusCell = document.createElement("td");
  const pill = document.createElement("span");
  pill.className = `status-pill pill-${status}`;
  const dot = document.createElement("i");
  const label = document.createElement("span");
  label.textContent = statusNames[status];
  pill.append(dot, label);
  statusCell.append(pill);
  row.append(statusCell);

  const actionCell = document.createElement("td");
  const action = document.createElement("button");
  action.className = "row-action";
  action.type = "button";
  action.dataset.rescan = "true";
  action.dataset.host = item.host;
  action.dataset.port = String(item.port);
  action.setAttribute("aria-label", `Volver a sondear ${target}`);
  action.textContent = "↻";
  actionCell.append(action);
  row.append(actionCell);
  return row;
}

function updateLiveSummary(items) {
  const critical = items.filter((item) => item.status === "critical" || item.status === "error").length;
  const warning = items.filter((item) => item.status === "warning").length;
  const findings = items.filter((item) => item.status === "critical" || item.status === "warning" || item.status === "error");
  document.querySelector("#cert-total").textContent = items.length;
  document.querySelector("#critical-total").textContent = String(critical).padStart(2, "0");
  document.querySelector("#warning-total").textContent = String(warning).padStart(2, "0");
  document.querySelector("#probe-state").textContent = items.length ? "Activo" : "Listo";
  document.querySelector("#probe-detail").textContent = items.length ? `${items.length} objetivos locales` : "añade tu primer objetivo";
  document.querySelector("#finding-total").textContent = String(findings.length).padStart(2, "0");
  document.querySelector(".heading-count").textContent = String(findings.length).padStart(2, "0");
  document.querySelector(".nav-count-alert").textContent = String(findings.length).padStart(2, "0");
  document.querySelector(".nav-item[data-view-button='certificates'] .nav-count").textContent = items.length;
  document.querySelector("#chart-total").textContent = critical + warning;
  document.querySelector("#attention-note").textContent = "Hallazgos calculados con los sondeos locales.";

  const ranges = {
    critical: items.filter((item) => !item.error && item.days_left <= 7).length,
    warning: items.filter((item) => !item.error && item.days_left > 7 && item.days_left <= 30).length,
    next: items.filter((item) => !item.error && item.days_left > 30 && item.days_left <= 60).length,
    later: items.filter((item) => !item.error && item.days_left > 60 && item.days_left <= 90).length,
  };
  const maxRange = Math.max(1, ...Object.values(ranges));
  Object.entries(ranges).forEach(([name, count]) => {
    document.querySelector(`[data-chart-value='${name}']`).textContent = count;
    document.querySelector(`[data-chart-bar='${name}']`).style.setProperty("--bar-height", `${Math.max(3, count / maxRange * 100)}%`);
  });
  const attentionList = document.querySelector("#attention-list");
  attentionList.replaceChildren();
  if (findings.length === 0) {
    const empty = document.createElement("li");
    empty.className = "no-findings";
    empty.textContent = "Sin hallazgos abiertos en los objetivos sondeados.";
    attentionList.append(empty);
  } else {
    findings.slice(0, 4).forEach((item) => {
      const entry = document.createElement("li");
      const marker = document.createElement("span");
      marker.className = `severity-mark ${item.status === "warning" ? "severity-warning" : "severity-critical"}`;
      const copy = document.createElement("div");
      const title = document.createElement("strong");
      title.textContent = item.error ? "No se pudo completar el sondeo" : item.status === "critical" ? "Certificado próximo a caducar" : "Certificado dentro del umbral de aviso";
      const detail = document.createElement("span");
      detail.textContent = `${displayTarget(item)} · ${item.error || `${item.days_left} días`}`;
      copy.append(title, detail);
      const severity = document.createElement("b");
      severity.textContent = item.status === "warning" ? "AVISO" : "CRÍTICO";
      entry.append(marker, copy, severity);
      attentionList.append(entry);
    });
  }
}

function renderLiveCertificates(items) {
  liveCertificates = items || [];
  rowsContainer.replaceChildren(...liveCertificates.map(createCertificateRow));
  document.querySelector("#empty-state").textContent = "Aún no hay objetivos. Usa “Añadir objetivo” para lanzar el primer sondeo TLS.";
  document.querySelector("#table-update-label").textContent = liveCertificates.length ? "Resultados del sondeo local" : "Sin objetivos todavía";
  updateLiveSummary(liveCertificates);
  applyFilters();
}

function showNotice(message, isError = false) {
  const notice = document.querySelector("#app-notice");
  notice.textContent = message;
  notice.classList.toggle("is-error", isError);
  notice.hidden = false;
  window.clearTimeout(showNotice.timer);
  showNotice.timer = window.setTimeout(() => { notice.hidden = true; }, 5000);
}

async function loadLiveCertificates() {
  const response = await fetch("/api/certificates", { cache: "no-store" });
  if (!response.ok) throw new Error("No se pudo leer el inventario local.");
  const data = await response.json();
  renderLiveCertificates(data.certificates);
}

async function probeTarget(host, port) {
  const response = await fetch("/api/check", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ host, port }),
  });
  const result = await response.json();
  if (!response.ok) throw new Error(result.error || "No se pudo iniciar el sondeo.");
  await loadLiveCertificates();
  if (result.error) showNotice(`Sondeo completado con error: ${result.error}`, true);
  else showNotice(`Certificado de ${displayTarget(result)} actualizado.`);
}

async function connectLocalAPI() {
  try {
    const response = await fetch("/api/healthz", { cache: "no-store" });
    if (!response.ok) return;
    localAPI = true;
    document.querySelector("#workspace-mode").textContent = "Sondeo en esta máquina";
    document.querySelector("#server-state").textContent = "Servidor local activo";
    document.querySelector("#server-detail").textContent = "API de Vigía conectada";
    document.querySelector("#server-dot").classList.add("is-live");
    document.querySelector("#data-mode").lastChild.textContent = " SONDEO LOCAL";
    document.querySelector("#probe-state").textContent = "Listo";
    document.querySelector("#probe-detail").textContent = "API local conectada";
    document.querySelector("#probe-dot").classList.add("dot-green");
    document.querySelector("#dialog-description").textContent = "Vigía se conectará a este host desde tu equipo para leer el certificado presentado.";
    document.querySelector("#submit-target").textContent = "Sondear y añadir";
    document.querySelector("#attention-note").textContent = "Los hallazgos se calculan con los sondeos locales.";
    document.querySelector(".page-footer span:last-child").textContent = "Sondeos bajo demanda · resultados reales";
    await loadLiveCertificates();
  } catch {
    localAPI = false;
  }
}

function setView(view) {
  activeView = view;
  body.dataset.view = view;
  const copy = pageCopy[view];
  document.querySelector("#breadcrumb-current").textContent = copy.breadcrumb;
  document.querySelector("#page-title").innerHTML = copy.title;
  document.querySelector("#page-description").textContent = copy.description;
  document.querySelector("#inventory-title").firstChild.textContent = `${copy.inventory} `;
  document.querySelectorAll("[data-view-button]").forEach((button) => {
    const active = button.dataset.viewButton === view;
    button.classList.toggle("is-active", active);
    if (active) button.setAttribute("aria-current", "page");
    else button.removeAttribute("aria-current");
  });
  applyFilters();
}

function matchesFilter(row) {
  const status = row.dataset.status;
  const days = Number(row.dataset.days);
  if (activeView === "findings" && status === "healthy") return false;
  if (activeFilter === "critical") return status === "critical";
  if (activeFilter === "warning") return status === "warning";
  if (activeFilter === "next") return days > 7 && days <= 60;
  return true;
}

function applyFilters() {
  const query = searchInput.value.trim().toLocaleLowerCase("es");
  let visible = 0;
  rowsContainer.querySelectorAll("tr").forEach((row) => {
    const matchesText = row.textContent.toLocaleLowerCase("es").includes(query);
    const show = matchesText && matchesFilter(row);
    row.hidden = !show;
    if (show) visible += 1;
  });
  document.querySelector("#visible-count").textContent = visible;
  document.querySelector("#footer-count").textContent = visible;
  document.querySelector("#empty-state").hidden = visible !== 0;
}

function setFilter(filter) {
  activeFilter = filter;
  filterButtons.forEach((button) => {
    const selected = button.dataset.filter === filter;
    button.classList.toggle("is-selected", selected);
    button.setAttribute("aria-pressed", String(selected));
  });
  applyFilters();
  document.querySelector("#inventory-section").scrollIntoView({ behavior: "smooth", block: "start" });
}

document.querySelectorAll("[data-view-button]").forEach((button) => {
  button.addEventListener("click", () => {
    setView(button.dataset.viewButton);
    body.classList.remove("nav-open");
    document.querySelector("#mobile-menu").setAttribute("aria-expanded", "false");
  });
});

filterButtons.forEach((button) => button.addEventListener("click", () => setFilter(button.dataset.filter)));
searchInput.addEventListener("input", applyFilters);

document.querySelectorAll("[data-chart-filter]").forEach((button) => {
  button.addEventListener("click", () => {
    setView("certificates");
    setFilter(button.dataset.chartFilter);
  });
});

document.querySelector("#show-certificates").addEventListener("click", () => setView("certificates"));
document.querySelector("#show-findings").addEventListener("click", () => setView("findings"));
document.querySelector("#open-target-dialog").addEventListener("click", () => dialog.showModal());
document.querySelector("#close-dialog").addEventListener("click", () => dialog.close());
document.querySelector("#cancel-dialog").addEventListener("click", () => dialog.close());
document.querySelector("#mobile-menu").addEventListener("click", (event) => {
  const open = body.classList.toggle("nav-open");
  event.currentTarget.setAttribute("aria-expanded", String(open));
});
document.querySelector("#mobile-scrim").addEventListener("click", () => {
  body.classList.remove("nav-open");
  document.querySelector("#mobile-menu").setAttribute("aria-expanded", "false");
});

document.querySelector("#target-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  const host = String(form.get("host")).trim();
  const port = Number(String(form.get("port")).trim() || "443");
  if (!host) return;
  if (localAPI) {
    const submit = document.querySelector("#submit-target");
    submit.disabled = true;
    submit.textContent = "Sondeando...";
    try {
      await probeTarget(host, port);
      event.currentTarget.reset();
      document.querySelector("#target-port").value = "443";
      dialog.close();
      setFilter("all");
      setView("certificates");
    } catch (error) {
      showNotice(error.message, true);
    } finally {
      submit.disabled = false;
      submit.textContent = "Sondear y añadir";
    }
    return;
  }
  const row = document.createElement("tr");
  row.dataset.status = "pending";
  row.dataset.days = "9999";
  row.innerHTML = `<td><span class="target-name"></span><span class="target-meta"></span></td><td>Sin sondear</td><td>Sin datos</td><td class="days-cell">—</td><td><span class="status-pill pill-pending"><i></i>Pendiente</span></td><td><button class="row-action" type="button" aria-label="Objetivo añadido">✓</button></td>`;
  row.querySelector(".target-name").textContent = `${host}:${port}`;
  row.querySelector(".target-meta").textContent = `${form.get("group")} · añadido a la muestra`;
  rowsContainer.prepend(row);
  document.querySelector("#cert-total").textContent = Number(document.querySelector("#cert-total").textContent) + 1;
  event.currentTarget.reset();
  document.querySelector("#target-port").value = "443";
  dialog.close();
  setFilter("all");
  setView("certificates");
});

rowsContainer.addEventListener("click", async (event) => {
  const button = event.target.closest("[data-rescan]");
  if (!button || !localAPI) return;
  button.disabled = true;
  try {
    await probeTarget(button.dataset.host, Number(button.dataset.port));
  } catch (error) {
    showNotice(error.message, true);
  } finally {
    button.disabled = false;
  }
});

document.querySelector("#export-csv").addEventListener("click", () => {
  const visibleRows = [...rowsContainer.querySelectorAll("tr")].filter((row) => !row.hidden);
  const csvRows = [["Objetivo", "Grupo", "Emisor", "Vencimiento", "Dias restantes", "Estado"]];
  visibleRows.forEach((row) => {
    const cells = row.querySelectorAll("td");
    csvRows.push([
      cells[0].querySelector(".target-name").textContent,
      cells[0].querySelector(".target-meta").textContent,
      cells[1].textContent,
      cells[2].textContent,
      row.dataset.days === "9999" ? "" : row.dataset.days,
      cells[4].textContent.trim(),
    ]);
  });
  const csv = csvRows.map((row) => row.map((cell) => `"${cell.replaceAll('"', '""')}"`).join(",")).join("\n");
  const url = URL.createObjectURL(new Blob(["\ufeff", csv], { type: "text/csv;charset=utf-8" }));
  const link = document.createElement("a");
  link.href = url;
  link.download = "vigia-inventario-demo.csv";
  document.body.append(link);
  link.click();
  link.remove();
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
});

document.addEventListener("keydown", (event) => {
  if (event.key === "/" && document.activeElement.tagName !== "INPUT") {
    event.preventDefault();
    searchInput.focus();
  }
});

applyFilters();
connectLocalAPI();