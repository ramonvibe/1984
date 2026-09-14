(() => {
  "use strict";
  const csrf = () => document.querySelector('meta[name="csrf-token"]')?.content || "";
  let noticeTimeout;
  function notify(message) {
    const notice = document.getElementById("notice");
    notice.textContent = message;
    notice.hidden = false;
    clearTimeout(noticeTimeout);
    noticeTimeout = setTimeout(() => { notice.hidden = true; }, 9000);
  }
  document.addEventListener("htmx:configRequest", event => {
    event.detail.headers["X-CSRF-Token"] = csrf();
  });
  document.addEventListener("htmx:responseError", event => {
    notify(event.detail.xhr.responseText.slice(0,300) || "Could not save. Try again.");
  });
  document.addEventListener("htmx:sendError", () => notify("Connection lost. Your change was not saved."));
  document.addEventListener("submit", event => {
    const message = event.target.dataset.confirm;
    if (message && !window.confirm(message)) event.preventDefault();
  });
  document.addEventListener("keydown", event => {
    if (event.key !== "/" || event.ctrlKey || event.metaKey || event.altKey ||
        /INPUT|TEXTAREA|SELECT/.test(document.activeElement.tagName) ||
        document.activeElement.isContentEditable) return;
    const search = document.querySelector('.topbar input[type="search"]');
    if (search) { event.preventDefault(); search.focus(); }
  });

  let dragged = null;
  document.addEventListener("dragstart", event => {
    const card = event.target.closest(".issue-card");
    if (!card || card.getAttribute("aria-busy") === "true") return;
    dragged = card;
    card.classList.add("dragging");
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", card.dataset.issue);
  });
  document.addEventListener("dragover", event => {
    const column = event.target.closest(".board-column");
    if (!dragged || !column) return;
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
    document.querySelectorAll(".drag-over").forEach(item => item.classList.remove("drag-over"));
    column.classList.add("drag-over");
  });
  document.addEventListener("dragend", () => {
    document.querySelectorAll(".dragging,.drag-over").forEach(item => item.classList.remove("dragging","drag-over"));
    dragged = null;
  });
  document.addEventListener("drop", async event => {
    const column = event.target.closest(".board-column");
    if (!dragged || !column) return;
    event.preventDefault();
    const card = dragged;
    const origin = card.closest(".board-column");
    if (column === origin) return;
    card.setAttribute("aria-busy","true");
    try {
      const response = await fetch("/issues/" + encodeURIComponent(card.dataset.issue) + "/status", {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded", "X-CSRF-Token": csrf(), "Accept": "application/json" },
        body: new URLSearchParams({ status: column.dataset.status }),
        redirect: "error"
      });
      if (!response.ok) throw new Error((await response.text()).slice(0,300) || "Could not move this issue.");
      column.querySelector(".board-cards").prepend(card);
      [origin,column].forEach(item => {
        item.querySelector(".column-count").textContent = item.querySelectorAll(".issue-card").length;
      });
    } catch (error) {
      notify(error.message || "Connection lost. The card has not moved.");
    } finally {
      card.removeAttribute("aria-busy");
      card.classList.remove("dragging");
      document.querySelectorAll(".drag-over").forEach(item => item.classList.remove("drag-over"));
    }
  });
})();
