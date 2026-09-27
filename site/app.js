(() => {
  const buttons = document.querySelectorAll("[data-copy-target]");

  async function copyText(text) {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text);
      return;
    }

    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.setAttribute("readonly", "");
    textarea.style.position = "fixed";
    textarea.style.opacity = "0";
    document.body.appendChild(textarea);
    textarea.select();
    const copied = document.execCommand("copy");
    textarea.remove();
    if (!copied) throw new Error("copy failed");
  }

  for (const button of buttons) {
    button.addEventListener("click", async () => {
      const target = document.getElementById(button.dataset.copyTarget);
      if (!target) return;

      try {
        await copyText(target.textContent);
        button.classList.add("copied");
        button.setAttribute("aria-label", "Copied");
        button.title = "Copied";
        window.setTimeout(() => {
          button.classList.remove("copied");
          button.setAttribute("aria-label", "Copy all Quick Start commands");
          button.title = "Copy all commands";
        }, 1600);
      } catch {
        button.setAttribute("aria-label", "Copy failed");
        button.title = "Copy failed";
      }
    });
  }
})();
