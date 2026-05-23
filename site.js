const copyButtons = document.querySelectorAll("[data-copy]");

for (const button of copyButtons) {
  button.addEventListener("click", async () => {
    const text = button.getAttribute("data-copy");
    if (!text) {
      return;
    }

    try {
      await navigator.clipboard.writeText(text);
      button.classList.add("copied");
      button.setAttribute("title", "Copied");
      window.setTimeout(() => {
        button.classList.remove("copied");
        button.setAttribute("title", "Copy command");
      }, 1400);
    } catch {
      button.setAttribute("title", "Copy failed");
    }
  });
}
