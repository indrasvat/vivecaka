const copyButtons = document.querySelectorAll("[data-copy]");
const screenshotButtons = document.querySelectorAll("[data-lightbox-src]");
const lightbox = document.querySelector(".lightbox");
const lightboxImage = lightbox?.querySelector("img");
const lightboxTitle = lightbox?.querySelector("#lightbox-title");
const lightboxCloseButtons = lightbox?.querySelectorAll(".lightbox-backdrop, .lightbox-close") ?? [];
let lastLightboxTrigger = null;
const copyTimers = new WeakMap();

async function copyText(text) {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.setAttribute("readonly", "");
    textarea.style.position = "fixed";
    textarea.style.top = "-999px";
    textarea.style.opacity = "0";
    document.body.append(textarea);
    textarea.select();
    textarea.setSelectionRange(0, textarea.value.length);

    try {
      return document.execCommand("copy");
    } catch {
      return false;
    } finally {
      textarea.remove();
    }
  }
}

function flashCopyState(button, state) {
  const timer = copyTimers.get(button);
  if (timer) {
    window.clearTimeout(timer);
  }
  const originalTitle = button.dataset.copyTitle ?? "";

  button.classList.remove("copied", "copy-failed");
  button.classList.add(state);
  button.setAttribute("aria-label", state === "copied" ? "Copied" : "Copy failed");
  button.removeAttribute("title");

  copyTimers.set(button, window.setTimeout(() => {
    button.classList.remove("copied", "copy-failed");
    button.setAttribute("aria-label", button.dataset.copyLabel ?? "Copy command");
    if (originalTitle) {
      button.setAttribute("title", originalTitle);
    }
    copyTimers.delete(button);
  }, 1500));
}

for (const button of copyButtons) {
  button.dataset.copyLabel = button.getAttribute("aria-label") ?? "Copy command";
  button.dataset.copyTitle = button.getAttribute("title") ?? "";

  button.addEventListener("click", async () => {
    const text = button.getAttribute("data-copy");
    if (!text) {
      return;
    }

    if (await copyText(text)) {
      flashCopyState(button, "copied");
    } else {
      flashCopyState(button, "copy-failed");
    }
  });
}

function closeLightbox() {
  if (!lightbox || !lightboxImage) {
    return;
  }

  lightbox.classList.remove("is-open");
  lightbox.setAttribute("aria-hidden", "true");
  document.body.classList.remove("lightbox-open");
  lightboxImage.removeAttribute("src");
  lightboxImage.alt = "";

  if (lastLightboxTrigger) {
    lastLightboxTrigger.focus();
  }
}

for (const button of screenshotButtons) {
  button.addEventListener("click", () => {
    if (!lightbox || !lightboxImage || !lightboxTitle) {
      return;
    }

    const src = button.getAttribute("data-lightbox-src");
    const title = button.getAttribute("data-lightbox-title") ?? "Screenshot";
    const image = button.querySelector("img");

    if (!src) {
      return;
    }

    lastLightboxTrigger = button;
    lightboxImage.src = src;
    lightboxImage.alt = image?.alt ?? title;
    lightboxTitle.textContent = title;
    lightbox.classList.add("is-open");
    lightbox.setAttribute("aria-hidden", "false");
    document.body.classList.add("lightbox-open");
    lightbox.querySelector(".lightbox-close")?.focus();
  });
}

for (const button of lightboxCloseButtons) {
  button.addEventListener("click", closeLightbox);
}

document.addEventListener("keydown", (event) => {
  if (event.key === "Escape" && lightbox?.classList.contains("is-open")) {
    closeLightbox();
  }
});
