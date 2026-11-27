(() => {
  "use strict";

  const $ = (selector) => document.querySelector(selector);
  const fieldsHost = $("#formFields");
  const qrPreview = $("#qrPreview");
  const emptyState = $("#emptyState");
  const loader = $("#previewLoader");
  const statusMessage = $("#statusMessage");
  const generateButton = $("#generateButton");
  const downloadButton = $("#downloadButton");
  const copyButton = $("#copyButton");
  const apiKeyInput = $("#apiKey");
  const keyStatus = $("#keyStatus");

  let activeType = "url";
  let currentDataUri = "";
  let lastRequest = null;
  let debounceTimer = null;
  let logoBase64 = "";

  const typeLabels = {
    url: "Website", text: "Text", vcard: "Contact", wifi: "Wi-Fi",
    calendar: "Event", email: "Email", sms: "SMS", number: "Number",
    geolocation: "Location", file: "Small file"
  };

  const formDefinitions = {
    url: [
      { name: "url", label: "Website URL", type: "url", placeholder: "https://example.com", required: true }
    ],
    text: [
      { name: "text", label: "Text", type: "textarea", placeholder: "Enter the text you want to encode", required: true }
    ],
    vcard: [
      { row: [
        { name: "first_name", label: "First name", type: "text", placeholder: "Ada" },
        { name: "last_name", label: "Last name", type: "text", placeholder: "Lovelace" }
      ]},
      { name: "organization", label: "Organization", type: "text", placeholder: "Analytical Engines" },
      { name: "title", label: "Job title", type: "text", placeholder: "Mathematician" },
      { row: [
        { name: "phone", label: "Phone", type: "tel", placeholder: "+44 1234 567890" },
        { name: "email", label: "Email", type: "email", placeholder: "ada@example.com" }
      ]},
      { name: "website", label: "Website", type: "url", placeholder: "https://example.com" },
      { name: "address", label: "Address", type: "text", placeholder: "London" }
    ],
    wifi: [
      { name: "ssid", label: "Network name", type: "text", placeholder: "Office-Guest", required: true },
      { name: "password", label: "Password", type: "password", placeholder: "Enter network password" },
      { row: [
        { name: "security", label: "Security", type: "select", options: [["WPA", "WPA"], ["WEP", "WEP"], ["NOPASS", "Open network"]], value: "WPA" },
        { name: "hidden", label: "Hidden network", type: "checkbox", checkboxLabel: "This network is hidden" }
      ]}
    ],
    calendar: [
      { name: "title", label: "Event title", type: "text", placeholder: "Project review", required: true },
      { name: "description", label: "Description", type: "textarea", placeholder: "Review the release" },
      { name: "location", label: "Location", type: "text", placeholder: "Conference Room A" },
      { name: "timezone", label: "Timezone", type: "text", placeholder: "Europe/London" },
      { name: "all_day", label: "All-day event", type: "checkbox", checkboxLabel: "Make this an all-day event" },
      { row: [
        { name: "start", label: "Starts", type: "datetime-local", required: true },
        { name: "end", label: "Ends", type: "datetime-local" }
      ]}
    ],
    email: [
      { name: "email", label: "Email address", type: "email", placeholder: "hello@example.com", required: true },
      { name: "subject", label: "Subject", type: "text", placeholder: "Hello" },
      { name: "body", label: "Message", type: "textarea", placeholder: "Please send me more information." }
    ],
    sms: [
      { name: "phone", label: "Phone number", type: "tel", placeholder: "+15551234567", required: true },
      { name: "body", label: "Message", type: "textarea", placeholder: "Hello" }
    ],
    number: [
      { name: "number", label: "Number", type: "text", placeholder: "+1 (555) 123-4567", required: true }
    ],
    geolocation: [
      { row: [
        { name: "latitude", label: "Latitude", type: "number", placeholder: "51.5074", step: "any", required: true },
        { name: "longitude", label: "Longitude", type: "number", placeholder: "-0.1278", step: "any", required: true }
      ]},
      { name: "label", label: "Location label", type: "text", placeholder: "London" }
    ],
    file: [
      { name: "filename", label: "Filename", type: "text", placeholder: "hello.txt", required: true },
      { name: "mime_type", label: "MIME type", type: "text", placeholder: "text/plain", value: "application/octet-stream" },
      { name: "file", label: "File", type: "file", accept: "*/*", required: true }
    ]
  };

  function escapeHTML(value) {
    return String(value).replace(/[&<>'"]/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" })[char]);
  }

  function fieldHTML(field) {
    const value = field.value ? ` value="${escapeHTML(field.value)}"` : "";
    const required = field.required ? " required" : "";
    const placeholder = field.placeholder ? ` placeholder="${escapeHTML(field.placeholder)}"` : "";

    if (field.type === "textarea") {
      return `<label class="field"><span>${escapeHTML(field.label)}</span><textarea name="${field.name}"${placeholder}${required}></textarea></label>`;
    }
    if (field.type === "select") {
      const options = field.options.map(([valueOption, label]) => `<option value="${escapeHTML(valueOption)}"${valueOption === field.value ? " selected" : ""}>${escapeHTML(label)}</option>`).join("");
      return `<label class="field"><span>${escapeHTML(field.label)}</span><select name="${field.name}">${options}</select></label>`;
    }
    if (field.type === "checkbox") {
      return `<label class="checkbox-row"><input name="${field.name}" type="checkbox"><span>${escapeHTML(field.checkboxLabel || field.label)}</span></label>`;
    }
    if (field.type === "file") {
      return `<label class="upload"><input name="${field.name}" type="file" accept="${escapeHTML(field.accept || "")}"${required}><span class="upload-icon">⇧</span><span><strong id="fileUploadName">Choose a file</strong><small>Maximum 1,024 bytes</small></span></label>`;
    }

    const step = field.step ? ` step="${escapeHTML(field.step)}"` : "";
    return `<label class="field"><span>${escapeHTML(field.label)}</span><input name="${field.name}" type="${field.type}"${placeholder}${value}${required}${step}></label>`;
  }

  function renderForm() {
    const html = formDefinitions[activeType].map((item) => {
      if (item.row) {
        return `<div class="field-row">${item.row.map(fieldHTML).join("")}</div>`;
      }
      return fieldHTML(item);
    }).join("");
    fieldsHost.innerHTML = html;

    if (activeType === "calendar") {
      const allDay = fieldsHost.querySelector('[name="all_day"]');
      allDay.addEventListener("change", updateCalendarFields);
      updateCalendarFields();
    }

    if (activeType === "file") {
      const fileInput = fieldsHost.querySelector('[name="file"]');
      fileInput.addEventListener("change", () => {
        $("#fileUploadName").textContent = fileInput.files[0] ? fileInput.files[0].name : "Choose a file";
        scheduleGenerate();
      });
    }

    fieldsHost.querySelectorAll("input, textarea, select").forEach((control) => {
      control.addEventListener("input", scheduleGenerate);
      control.addEventListener("change", scheduleGenerate);
    });
  }

  function updateCalendarFields() {
    const allDay = fieldsHost.querySelector('[name="all_day"]')?.checked;
    fieldsHost.querySelectorAll('[name="start"], [name="end"]').forEach((input) => {
      input.type = allDay ? "date" : "datetime-local";
      input.value = "";
    });
  }

  function collectValues() {
    const values = { size: Number($("#size").value), recovery: $("#recovery").value };
    fieldsHost.querySelectorAll("[name]").forEach((control) => {
      if (control.type === "file") return;
      if (control.type === "checkbox") values[control.name] = control.checked;
      else if (control.value !== "") {
        if (control.type === "number") values[control.name] = Number(control.value);
        else values[control.name] = control.value;
      }
    });

    if ($("#showLogo").checked) {
      values.show_logo = true;
      if (logoBase64) values.logo_png_base64 = logoBase64;
    }

    if (activeType === "file") {
      values.mime_type = values.mime_type || "application/octet-stream";
    }
    return values;
  }

  function missingRequiredField() {
    const requiredControls = fieldsHost.querySelectorAll("[required]");
    for (const control of requiredControls) {
      if (control.type === "file") {
        if (!control.files || !control.files[0]) return control.name === "file" ? "Choose a file" : fieldLabel(control);
      } else if (!String(control.value).trim()) {
        return fieldLabel(control);
      }
    }

    if (activeType === "vcard") {
      const first = fieldsHost.querySelector('[name="first_name"]')?.value.trim();
      const last = fieldsHost.querySelector('[name="last_name"]')?.value.trim();
      if (!first && !last) return "First or last name";
    }
    if (activeType === "wifi") {
      const security = fieldsHost.querySelector('[name="security"]')?.value;
      const password = fieldsHost.querySelector('[name="password"]')?.value;
      if (security !== "NOPASS" && !password) return "Wi-Fi password";
    }
    if (activeType === "geolocation") {
      const latitude = fieldsHost.querySelector('[name="latitude"]')?.value;
      const longitude = fieldsHost.querySelector('[name="longitude"]')?.value;
      if (!latitude || !longitude) return "Latitude and longitude";
    }
    return "";
  }

  function fieldLabel(control) {
    return control.closest(".field")?.querySelector("span")?.textContent || control.name;
  }

  async function prepareFilePayload(values) {
    const fileInput = fieldsHost.querySelector('[name="file"]');
    const file = fileInput?.files[0];
    if (!file) return true;
    if (file.size > 1024) {
      setStatus("File payloads are limited to 1,024 bytes", "error");
      return false;
    }
    values.filename = values.filename || file.name;
    values.base64_content = await fileToBase64(file);
    return true;
  }

  function fileToBase64(file) {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(String(reader.result).split(",", 2)[1]);
      reader.onerror = () => reject(reader.error);
      reader.readAsDataURL(file);
    });
  }

  function fileToLogoBase64(file) {
    return new Promise((resolve, reject) => {
      if (file.size > 1024 * 1024) {
        reject(new Error("Logo exceeds 1 MB"));
        return;
      }
      const reader = new FileReader();
      reader.onload = () => resolve(String(reader.result).split(",", 2)[1]);
      reader.onerror = () => reject(reader.error);
      reader.readAsDataURL(file);
    });
  }

  async function generate(manual = false) {
    const key = apiKeyInput.value.trim();
    if (!key) {
      setStatus("Enter and save an API key to generate", "error");
      keyStatus.className = "status-dot error";
      return;
    }

    const missing = missingRequiredField();
    if (missing) {
      if (manual) setStatus(`${missing} is required`, "error");
      return;
    }

    const values = collectValues();
    if (activeType === "file") {
      const prepared = await prepareFilePayload(values);
      if (!prepared) return;
    }

    if (lastRequest) lastRequest.abort();
    lastRequest = new AbortController();

    setLoading(true);
    setStatus("Generating QR code…");

    try {
      const response = await fetch(`/api/v1/qr/${activeType}`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-API-Key": key },
        body: JSON.stringify(values),
        signal: lastRequest.signal
      });
      const body = await response.json();
      if (!response.ok) {
        throw new Error(body.error?.message || `Request failed (${response.status})`);
      }

      currentDataUri = body.qr_code.data_uri;
      qrPreview.src = currentDataUri;
      qrPreview.hidden = false;
      emptyState.hidden = true;
      downloadButton.disabled = false;
      copyButton.disabled = false;
      $("#previewMeta").textContent = `${body.qr_code.size} × ${body.qr_code.size} px`;
      setStatus("Ready — click download to save", "success");
    } catch (error) {
      if (error.name === "AbortError") return;
      setStatus(error.message, "error");
    } finally {
      setLoading(false);
    }
  }

  function scheduleGenerate() {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => generate(false), 550);
  }

  function setLoading(loading) {
    loader.hidden = !loading;
    generateButton.disabled = loading;
  }

  function setStatus(message, kind = "") {
    statusMessage.textContent = message;
    statusMessage.className = `status-message${kind ? ` ${kind}` : ""}`;
  }

  function downloadQR() {
    if (!currentDataUri) return;
    const link = document.createElement("a");
    link.href = currentDataUri;
    link.download = `qr-${activeType}-${Date.now()}.png`;
    document.body.appendChild(link);
    link.click();
    link.remove();
  }

  async function copyQR() {
    if (!currentDataUri) return;
    try {
      const response = await fetch(currentDataUri);
      const blob = await response.blob();
      await navigator.clipboard.write([new ClipboardItem({ "image/png": blob })]);
      setStatus("QR code copied to clipboard", "success");
    } catch (_) {
      setStatus("Clipboard image copying is not supported here", "error");
    }
  }

  document.querySelectorAll(".type-button").forEach((button) => {
    button.addEventListener("click", () => {
      document.querySelectorAll(".type-button").forEach((item) => item.classList.remove("active"));
      button.classList.add("active");
      activeType = button.dataset.type;
      $("#typeTitle").textContent = typeLabels[activeType];
      qrPreview.hidden = true;
      qrPreview.removeAttribute("src");
      emptyState.hidden = false;
      currentDataUri = "";
      downloadButton.disabled = true;
      copyButton.disabled = true;
      setStatus("Choose the details for your QR code");
      renderForm();
    });
  });

  $("#size").addEventListener("input", (event) => {
    $("#sizeOutput").textContent = `${event.target.value} px`;
    $("#previewMeta").textContent = `${event.target.value} × ${event.target.value} px`;
  });

  $("#showLogo").addEventListener("change", (event) => {
    $("#logoUploadWrap").classList.toggle("hidden", !event.target.checked);
    scheduleGenerate();
  });

  $("#logoUpload").addEventListener("change", async (event) => {
    const file = event.target.files[0];
    if (!file) {
      logoBase64 = "";
      $("#logoUploadName").textContent = "Choose a PNG logo";
      scheduleGenerate();
      return;
    }
    try {
      logoBase64 = await fileToLogoBase64(file);
      $("#logoUploadName").textContent = file.name;
      scheduleGenerate();
    } catch (error) {
      logoBase64 = "";
      event.target.value = "";
      $("#logoUploadName").textContent = "Choose a PNG logo";
      setStatus(error.message, "error");
    }
  });

  $("#saveKey").addEventListener("click", () => {
    const key = apiKeyInput.value.trim();
    if (!key) {
      sessionStorage.removeItem("qrStudioApiKey");
      keyStatus.className = "status-dot";
      setStatus("API key cleared", "error");
      return;
    }
    sessionStorage.setItem("qrStudioApiKey", key);
    keyStatus.className = "status-dot online";
    keyStatus.title = "API key saved for this tab";
    setStatus("API key saved — ready to generate", "success");
    generate(false);
  });

  apiKeyInput.addEventListener("keydown", (event) => {
    if (event.key === "Enter") $("#saveKey").click();
  });

  generateButton.addEventListener("click", () => generate(true));
  downloadButton.addEventListener("click", downloadQR);
  copyButton.addEventListener("click", copyQR);

  const savedKey = sessionStorage.getItem("qrStudioApiKey");
  if (savedKey) {
    apiKeyInput.value = savedKey;
    keyStatus.className = "status-dot online";
    keyStatus.title = "API key saved for this tab";
  }

  renderForm();
})();
