const examples = {
  curl: `$ curl --request POST http://localhost:8080/api/v1/qr/url \\
  --header 'X-API-Key: your-api-key' \\
  --header 'Content-Type: application/json' \\
  --data '{"url":"https://example.com","show_logo":true}'`,
  javascript: `const response = await fetch("http://localhost:8080/api/v1/qr/url", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "X-API-Key": "your-api-key"
  },
  body: JSON.stringify({
    url: "https://example.com",
    show_logo: true
  })
});

const { qr_code } = await response.json();
console.log(qr_code.data_uri);`,
  python: `import requests

response = requests.post(
    "http://localhost:8080/api/v1/qr/url",
    headers={
        "X-API-Key": "your-api-key",
        "Content-Type": "application/json",
    },
    json={
        "url": "https://example.com",
        "show_logo": True,
    },
)

qr_code = response.json()["qr_code"]
print(qr_code["data_uri"])`
};

const escapeHtml = (value) => value
  .replace(/&/g, "&amp;")
  .replace(/</g, "&lt;")
  .replace(/>/g, "&gt;")
  .replace(/"/g, "&quot;")
  .replace(/'/g, "&#039;");

const renderExample = (name) => {
  const code = document.querySelector("#code-example");
  if (!code) return;

  if (name === "curl") {
    code.innerHTML = `<span class="prompt">$</span> curl --request POST http://localhost:8080/api/v1/qr/url \\
  --header <span class="string">&#39;X-API-Key: your-api-key&#39;</span> \\
  --header <span class="string">&#39;Content-Type: application/json&#39;</span> \\
  --data <span class="string">&#39;{"url":"https://example.com","show_logo":true}&#39;</span>`;
  } else {
    code.innerHTML = escapeHtml(examples[name]);
  }
};

const revealElements = document.querySelectorAll(".reveal");
if ("IntersectionObserver" in window) {
  const observer = new IntersectionObserver((entries) => {
    entries.forEach((entry) => {
      if (entry.isIntersecting) {
        entry.target.classList.add("visible");
        observer.unobserve(entry.target);
      }
    });
  }, { threshold: 0.12 });
  revealElements.forEach((element) => observer.observe(element));
} else {
  revealElements.forEach((element) => element.classList.add("visible"));
}

const menuButton = document.querySelector(".menu-toggle");
const navLinks = document.querySelector(".nav-links");

if (menuButton && navLinks) {
  menuButton.addEventListener("click", () => {
    const open = menuButton.getAttribute("aria-expanded") === "true";
    menuButton.setAttribute("aria-expanded", String(!open));
    menuButton.setAttribute("aria-label", open ? "Open navigation" : "Close navigation");
    navLinks.classList.toggle("open", !open);
    document.body.classList.toggle("menu-open", !open);
  });

  navLinks.querySelectorAll("a").forEach((link) => {
    link.addEventListener("click", () => {
      menuButton.setAttribute("aria-expanded", "false");
      menuButton.setAttribute("aria-label", "Open navigation");
      navLinks.classList.remove("open");
      document.body.classList.remove("menu-open");
    });
  });
}

let activeExample = "curl";
document.querySelectorAll(".code-tab").forEach((tab) => {
  tab.addEventListener("click", () => {
    activeExample = tab.dataset.example;
    document.querySelectorAll(".code-tab").forEach((item) => {
      const isActive = item === tab;
      item.classList.toggle("active", isActive);
      item.setAttribute("aria-selected", String(isActive));
    });
    renderExample(activeExample);
  });
});

const copyButton = document.querySelector(".copy-button");
if (copyButton) {
  copyButton.addEventListener("click", async () => {
    const label = copyButton.querySelector("span");
    try {
      await navigator.clipboard.writeText(examples[activeExample]);
      label.textContent = "Copied";
      copyButton.setAttribute("aria-label", "Code copied");
    } catch (_) {
      label.textContent = "Unavailable";
    }
    window.setTimeout(() => {
      label.textContent = "Copy";
      copyButton.setAttribute("aria-label", "Copy code");
    }, 1600);
  });
}
