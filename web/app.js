const body = document.body;
const rowsContainer = document.querySelector("#certificate-rows");
const searchInput = document.querySelector("#search-input");
const filterButtons = [...document.querySelectorAll(".filter-tab")];
const dialog = document.querySelector("#target-dialog");
let activeFilter = "all";
let activeView = "overview";

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

document.querySelector("#target-form").addEventListener("submit", (event) => {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  const host = String(form.get("host")).trim();
  const port = String(form.get("port")).trim() || "443";
  if (!host) return;
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