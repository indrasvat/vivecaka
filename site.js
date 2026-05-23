const copyButtons = document.querySelectorAll("[data-copy]");

for (const button of copyButtons) {
  button.addEventListener("click", async () => {
    const text = button.getAttribute("data-copy");
    const label = button.querySelector("span");
    if (!text || !label) {
      return;
    }

    try {
      await navigator.clipboard.writeText(text);
      const original = label.textContent;
      label.textContent = "Copied";
      window.setTimeout(() => {
        label.textContent = original;
      }, 1400);
    } catch {
      label.textContent = "Select";
    }
  });
}
