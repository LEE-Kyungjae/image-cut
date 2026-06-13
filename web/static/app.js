const canvas = document.querySelector("#previewCanvas");
const ctx = canvas.getContext("2d");
const fileInput = document.querySelector("#imageInput");
const controls = [...document.querySelectorAll("[data-grid-control]")];
const presetButtons = [...document.querySelectorAll("[data-preset]")];
const adjustButtons = [...document.querySelectorAll("[data-adjust]")];
const sampleButton = document.querySelector("#sampleButton");
const generateButton = document.querySelector("#generateButton");
const providerInput = document.querySelector("#providerInput");
const promptInput = document.querySelector("#promptInput");
const modelInput = document.querySelector("#modelInput");
const sizeInput = document.querySelector("#sizeInput");
const qualityInput = document.querySelector("#qualityInput");
const confirmInput = document.querySelector("#confirmInput");
const openaiFields = document.querySelector("#openaiFields");
const generateNote = document.querySelector("#generateNote");
const cropRectsInput = document.querySelector("#cropRectsInput");
const placeholder = document.querySelector("#placeholder");
const sourceMetric = document.querySelector("#sourceMetric");
const cellMetric = document.querySelector("#cellMetric");
const countMetric = document.querySelector("#countMetric");
const selectedMetric = document.querySelector("#selectedMetric");
const requestMetric = document.querySelector("#requestMetric");
const outputMetric = document.querySelector("#outputMetric");
const costMetric = document.querySelector("#costMetric");
const notice = document.querySelector("#notice");

let loadedImage = null;
let loadedUrl = "";
let selectedKey = "";
let adjustments = new Map();
let lastCells = [];
let pointerDrag = null;

fileInput.addEventListener("change", () => {
  const file = fileInput.files && fileInput.files[0];
  if (!file) {
    setImage(null);
    return;
  }

  if (loadedUrl) URL.revokeObjectURL(loadedUrl);
  loadedUrl = URL.createObjectURL(file);

  const img = new Image();
  img.onload = () => {
    loadedImage = img;
    placeholder.hidden = true;
    draw();
  };
  img.onerror = () => {
    setImage(null);
    notice.textContent = "이미지를 불러올 수 없습니다. PNG 또는 JPEG 파일을 선택하세요.";
  };
  img.src = loadedUrl;
});

controls.forEach((control) => {
  control.addEventListener("input", () => {
    adjustments = new Map();
    selectedKey = "";
    draw();
    syncCostPanel();
  });
});

presetButtons.forEach((button) => {
  button.addEventListener("click", () => {
    const [rows, cols] = button.dataset.preset.split("x").map((value) => Number.parseInt(value, 10));
    setField("rows", rows);
    setField("cols", cols);
    adjustments = new Map();
    selectedKey = "";
    draw();
    syncCostPanel();
  });
});

adjustButtons.forEach((button) => {
  button.addEventListener("click", () => adjustSelected(button.dataset.adjust));
});

canvas.addEventListener("pointerdown", (event) => {
  if (!loadedImage) return;
  const point = canvasPoint(event);
  const hit = hitTest(point.x, point.y);
  selectedKey = hit ? hit.cell.key : "";
  if (!hit) {
    pointerDrag = null;
    draw();
    return;
  }

  canvas.setPointerCapture(event.pointerId);
  pointerDrag = {
    pointerId: event.pointerId,
    key: hit.cell.key,
    mode: hit.mode,
    startX: point.x,
    startY: point.y,
    startAdjustment: { ...(adjustments.get(hit.cell.key) || { dx: 0, dy: 0, dw: 0, dh: 0 }) },
    scaleX: loadedImage.width / hit.imageRect.w,
    scaleY: loadedImage.height / hit.imageRect.h,
  };
  draw();
});

canvas.addEventListener("pointermove", (event) => {
  if (!pointerDrag || event.pointerId !== pointerDrag.pointerId) {
    updateCursor(event);
    return;
  }

  const point = canvasPoint(event);
  const dx = Math.round((point.x - pointerDrag.startX) * pointerDrag.scaleX);
  const dy = Math.round((point.y - pointerDrag.startY) * pointerDrag.scaleY);
  const next = { ...pointerDrag.startAdjustment };
  if (pointerDrag.mode === "resize") {
    next.dw += dx;
    next.dh += dy;
  } else {
    next.dx += dx;
    next.dy += dy;
  }
  adjustments.set(pointerDrag.key, next);
  draw();
});

canvas.addEventListener("pointerup", finishPointerDrag);
canvas.addEventListener("pointercancel", finishPointerDrag);
canvas.addEventListener("pointerleave", updateCursor);

sampleButton.addEventListener("click", async () => {
  const file = await createSampleFile(readOptions());
  loadFile(file);
});

providerInput.addEventListener("change", syncProviderUI);
sizeInput.addEventListener("change", syncCostPanel);
qualityInput.addEventListener("change", syncCostPanel);

generateButton.addEventListener("click", async () => {
  generateButton.disabled = true;
  generateButton.textContent = "생성 중";
  try {
    const file = await generateImageFile();
    loadFile(file);
  } catch (error) {
    notice.textContent = error instanceof Error ? error.message : "Mock 이미지를 생성할 수 없습니다.";
  } finally {
    generateButton.disabled = false;
    generateButton.textContent = "생성";
  }
});

window.addEventListener("resize", draw);
syncProviderUI();
draw();

function setImage(img) {
  loadedImage = img;
  adjustments = new Map();
  selectedKey = "";
  pointerDrag = null;
  placeholder.hidden = Boolean(img);
  draw();
}

function loadFile(file) {
  const transfer = new DataTransfer();
  transfer.items.add(file);
  fileInput.files = transfer.files;
  fileInput.dispatchEvent(new Event("change", { bubbles: true }));
}

function readOptions() {
  const data = new FormData(document.querySelector("#cutForm"));
  const rows = clampInt(data.get("rows"), 1, 8, 3);
  const cols = clampInt(data.get("cols"), 1, 8, 3);
  const margin = Math.max(0, clampInt(data.get("margin"), 0, 100000, 0));
  const gutter = Math.max(0, clampInt(data.get("gutter"), 0, 100000, 0));
  return { rows, cols, margin, gutter };
}

function clampInt(value, min, max, fallback) {
  const parsed = Number.parseInt(value, 10);
  if (!Number.isFinite(parsed)) return fallback;
  return Math.min(max, Math.max(min, parsed));
}

function setField(name, value) {
  const input = document.querySelector(`[name="${name}"]`);
  if (input) input.value = String(value);
}

function draw() {
  const rect = canvas.getBoundingClientRect();
  const scale = window.devicePixelRatio || 1;
  canvas.width = Math.max(1, Math.round(rect.width * scale));
  canvas.height = Math.max(1, Math.round(rect.height * scale));
  ctx.setTransform(scale, 0, 0, scale, 0, 0);
  ctx.clearRect(0, 0, rect.width, rect.height);

  const opts = readOptions();
  countMetric.textContent = String(opts.rows * opts.cols);

  if (!loadedImage) {
    sourceMetric.textContent = "-";
    cellMetric.textContent = "-";
    selectedMetric.textContent = "-";
    cropRectsInput.value = "";
    lastCells = [];
    return;
  }

  const imageRect = containRect(loadedImage.width, loadedImage.height, rect.width, rect.height);
  ctx.drawImage(loadedImage, imageRect.x, imageRect.y, imageRect.w, imageRect.h);

  const grid = calculateGrid(loadedImage.width, loadedImage.height, opts);
  sourceMetric.textContent = `${loadedImage.width} x ${loadedImage.height}`;

  if (!grid.valid) {
    cellMetric.textContent = "계산 불가";
    selectedMetric.textContent = "-";
    cropRectsInput.value = "";
    lastCells = [];
    notice.textContent = "margin/gutter 값이 이미지 크기보다 큽니다.";
    drawInvalidOverlay(imageRect);
    return;
  }

  const cells = buildCells(imageRect, loadedImage, grid, opts);
  lastCells = cells;
  if (selectedKey && !cells.some((cell) => cell.key === selectedKey)) {
    selectedKey = "";
  }

  cellMetric.textContent = `${grid.cellW} x ${grid.cellH}`;
  selectedMetric.textContent = selectedKey ? selectedKey.replace(",", " / ") : "캔버스에서 선택";
  cropRectsInput.value = JSON.stringify(cells.map((cell) => cell.rect));
  notice.textContent = makeNotice(opts, grid);
  drawGrid(cells);
}

function containRect(srcW, srcH, dstW, dstH) {
  const ratio = Math.min(dstW / srcW, dstH / srcH);
  const w = srcW * ratio;
  const h = srcH * ratio;
  return {
    x: (dstW - w) / 2,
    y: (dstH - h) / 2,
    w,
    h,
  };
}

function calculateGrid(width, height, opts) {
  const usableW = width - opts.margin * 2 - opts.gutter * (opts.cols - 1);
  const usableH = height - opts.margin * 2 - opts.gutter * (opts.rows - 1);
  if (usableW <= 0 || usableH <= 0) {
    return { valid: false };
  }

  const cellW = Math.floor(usableW / opts.cols);
  const cellH = Math.floor(usableH / opts.rows);
  if (cellW <= 0 || cellH <= 0) {
    return { valid: false };
  }

  return { valid: true, cellW, cellH };
}

function buildCells(imageRect, img, grid, opts) {
  const sx = imageRect.w / img.width;
  const sy = imageRect.h / img.height;
  const cells = [];
  const maxW = img.width;
  const maxH = img.height;

  for (let row = 0; row < opts.rows; row++) {
    for (let col = 0; col < opts.cols; col++) {
      const key = `${row},${col}`;
      const adjustment = adjustments.get(key) || { dx: 0, dy: 0, dw: 0, dh: 0 };
      const baseX = opts.margin + col * (grid.cellW + opts.gutter);
      const baseY = opts.margin + row * (grid.cellH + opts.gutter);
      const rect = clampRect({
        row,
        col,
        x: baseX + adjustment.dx,
        y: baseY + adjustment.dy,
        w: grid.cellW + adjustment.dw,
        h: grid.cellH + adjustment.dh,
      }, maxW, maxH);
      cells.push({
        key,
        rect,
        view: {
          x: imageRect.x + rect.x * sx,
          y: imageRect.y + rect.y * sy,
          w: rect.w * sx,
          h: rect.h * sy,
        },
      });
    }
  }

  return cells;
}

function clampRect(rect, maxW, maxH) {
  const w = Math.max(8, Math.min(rect.w, maxW));
  const h = Math.max(8, Math.min(rect.h, maxH));
  const x = Math.max(0, Math.min(rect.x, maxW - w));
  const y = Math.max(0, Math.min(rect.y, maxH - h));
  return { row: rect.row, col: rect.col, x, y, w, h };
}

function drawGrid(cells) {
  ctx.save();
  ctx.lineWidth = 2;
  ctx.strokeStyle = "#14b8a6";
  ctx.fillStyle = "rgba(20, 184, 166, 0.1)";
  ctx.font = "600 12px system-ui, sans-serif";
  ctx.textBaseline = "top";

  for (const cell of cells) {
    const selected = cell.key === selectedKey;
    ctx.lineWidth = selected ? 4 : 2;
    ctx.strokeStyle = selected ? "#f97316" : "#14b8a6";
    ctx.fillStyle = selected ? "rgba(249, 115, 22, 0.14)" : "rgba(20, 184, 166, 0.1)";
    ctx.fillRect(cell.view.x, cell.view.y, cell.view.w, cell.view.h);
    ctx.strokeRect(cell.view.x, cell.view.y, cell.view.w, cell.view.h);
    ctx.fillStyle = selected ? "rgba(194, 65, 12, 0.9)" : "rgba(15, 118, 110, 0.85)";
    ctx.fillText(`${cell.rect.row + 1},${cell.rect.col + 1}`, cell.view.x + 8, cell.view.y + 8);
    if (selected) {
      drawResizeHandle(cell.view);
    }
  }

  ctx.restore();
}

function drawResizeHandle(view) {
  const size = handleSize(view);
  ctx.fillStyle = "#f97316";
  ctx.strokeStyle = "#ffffff";
  ctx.lineWidth = 2;
  ctx.fillRect(view.x + view.w - size, view.y + view.h - size, size, size);
  ctx.strokeRect(view.x + view.w - size, view.y + view.h - size, size, size);
}

function handleSize(view) {
  return Math.max(12, Math.min(22, Math.min(view.w, view.h) * 0.12));
}

function adjustSelected(action) {
  if (!selectedKey) {
    notice.textContent = "먼저 캔버스에서 보정할 셀을 선택하세요.";
    return;
  }
  const step = 8;
  const current = adjustments.get(selectedKey) || { dx: 0, dy: 0, dw: 0, dh: 0 };
  const next = { ...current };

  switch (action) {
    case "up":
      next.dy -= step;
      break;
    case "down":
      next.dy += step;
      break;
    case "left":
      next.dx -= step;
      break;
    case "right":
      next.dx += step;
      break;
    case "grow":
      next.dx -= step;
      next.dy -= step;
      next.dw += step * 2;
      next.dh += step * 2;
      break;
    case "shrink":
      next.dx += step;
      next.dy += step;
      next.dw -= step * 2;
      next.dh -= step * 2;
      break;
    case "reset":
      adjustments.delete(selectedKey);
      draw();
      return;
  }

  adjustments.set(selectedKey, next);
  draw();
}

function finishPointerDrag(event) {
  if (!pointerDrag || event.pointerId !== pointerDrag.pointerId) return;
  if (canvas.hasPointerCapture(event.pointerId)) {
    canvas.releasePointerCapture(event.pointerId);
  }
  pointerDrag = null;
  updateCursor(event);
}

function canvasPoint(event) {
  const rect = canvas.getBoundingClientRect();
  return {
    x: event.clientX - rect.left,
    y: event.clientY - rect.top,
  };
}

function hitTest(x, y) {
  const imageRect = currentImageRect();
  if (!imageRect) return null;
  for (let i = lastCells.length - 1; i >= 0; i--) {
    const cell = lastCells[i];
    if (x < cell.view.x || x > cell.view.x + cell.view.w || y < cell.view.y || y > cell.view.y + cell.view.h) {
      continue;
    }
    const size = handleSize(cell.view);
    const inHandle = x >= cell.view.x + cell.view.w - size && y >= cell.view.y + cell.view.h - size;
    return { cell, mode: inHandle ? "resize" : "move", imageRect };
  }
  return null;
}

function currentImageRect() {
  if (!loadedImage) return null;
  const rect = canvas.getBoundingClientRect();
  return containRect(loadedImage.width, loadedImage.height, rect.width, rect.height);
}

function updateCursor(event) {
  if (!loadedImage || pointerDrag) return;
  const point = canvasPoint(event);
  const hit = hitTest(point.x, point.y);
  if (!hit) {
    canvas.style.cursor = "default";
    return;
  }
  canvas.style.cursor = hit.mode === "resize" ? "nwse-resize" : "move";
}

function drawInvalidOverlay(imageRect) {
  ctx.save();
  ctx.fillStyle = "rgba(180, 35, 24, 0.16)";
  ctx.fillRect(imageRect.x, imageRect.y, imageRect.w, imageRect.h);
  ctx.strokeStyle = "#b42318";
  ctx.lineWidth = 3;
  ctx.strokeRect(imageRect.x + 1, imageRect.y + 1, imageRect.w - 2, imageRect.h - 2);
  ctx.restore();
}

function makeNotice(opts, grid) {
  const minSide = Math.min(grid.cellW, grid.cellH);
  if (minSide < 256) {
    return "컷의 짧은 변이 256px 미만입니다. 결과가 작게 느껴질 수 있습니다.";
  }
  if (opts.rows >= 4 || opts.cols >= 4) {
    return "4칸 이상 그리드는 시안용으로 적합합니다. 마음에 드는 컷은 나중에 단일 고품질 재생성을 권장합니다.";
  }
  return "현재 설정은 안정적인 컷 크기입니다.";
}

async function createSampleFile(opts) {
  const size = 1200;
  const canvas = document.createElement("canvas");
  canvas.width = size;
  canvas.height = size;
  const sampleCtx = canvas.getContext("2d");

  sampleCtx.fillStyle = "#f8fafc";
  sampleCtx.fillRect(0, 0, size, size);

  const grid = calculateGrid(size, size, opts);
  if (!grid.valid) {
    setField("margin", 24);
    setField("gutter", 24);
    opts = readOptions();
  }

  drawSampleGrid(sampleCtx, size, opts);

  const blob = await new Promise((resolve) => canvas.toBlob(resolve, "image/png"));
  return new File([blob], `imagecut_sample_${opts.rows}x${opts.cols}.png`, { type: "image/png" });
}

async function generateImageFile() {
  const opts = readOptions();
  const params = new URLSearchParams({
    provider: providerInput.value,
    prompt: promptInput.value,
    rows: String(opts.rows),
    cols: String(opts.cols),
    margin: String(opts.margin),
    gutter: String(opts.gutter),
    model: modelInput.value,
    size: sizeInput.value,
    quality: qualityInput.value,
    openai_confirm: confirmInput.value,
  });
  const response = await fetch("/generate", {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: params,
  });
  if (!response.ok) {
    throw new Error((await response.text()).trim());
  }

  const blob = await response.blob();
  return new File([blob], `imagecut_${providerInput.value}_${opts.rows}x${opts.cols}.png`, { type: "image/png" });
}

function syncProviderUI() {
  const openai = providerInput.value === "openai";
  openaiFields.hidden = !openai;
  generateNote.textContent = openai
    ? "OpenAI는 서버에서 IMAGECUT_OPENAI_ENABLED=true, OPENAI_API_KEY, ALLOW_COST가 모두 맞아야 호출됩니다."
    : "Mock은 OpenAI API를 호출하지 않습니다.";
  syncCostPanel();
}

function syncCostPanel() {
  const opts = readOptions();
  const openai = providerInput.value === "openai";
  const cuts = opts.rows * opts.cols;
  requestMetric.textContent = "1 grid image";
  outputMetric.textContent = `${cuts} cuts from ${sizeInput.value}`;
  if (!openai) {
    costMetric.textContent = "무료 mock";
    return;
  }
  const megapixels = outputMegapixels(sizeInput.value);
  costMetric.textContent = `실비 과금: GPT-Image-2 image output $30/1M tokens, ${qualityInput.value}, ${megapixels}MP`;
}

function outputMegapixels(size) {
  const [w, h] = size.split("x").map((value) => Number.parseInt(value, 10));
  if (!Number.isFinite(w) || !Number.isFinite(h)) return "auto";
  return ((w * h) / 1_000_000).toFixed(2);
}

function drawSampleGrid(sampleCtx, size, opts) {
  const palette = [
    ["#0f766e", "#ccfbf1"],
    ["#7c3aed", "#ede9fe"],
    ["#c2410c", "#ffedd5"],
    ["#1d4ed8", "#dbeafe"],
    ["#be123c", "#ffe4e6"],
    ["#047857", "#d1fae5"],
    ["#a16207", "#fef3c7"],
    ["#4338ca", "#e0e7ff"],
  ];
  const grid = calculateGrid(size, size, opts);

  sampleCtx.fillStyle = "#ffffff";
  sampleCtx.fillRect(0, 0, size, size);
  sampleCtx.font = "700 64px system-ui, sans-serif";
  sampleCtx.textAlign = "center";
  sampleCtx.textBaseline = "middle";

  for (let row = 0; row < opts.rows; row++) {
    for (let col = 0; col < opts.cols; col++) {
      const index = row * opts.cols + col;
      const [ink, fill] = palette[index % palette.length];
      const x = opts.margin + col * (grid.cellW + opts.gutter);
      const y = opts.margin + row * (grid.cellH + opts.gutter);
      sampleCtx.fillStyle = fill;
      sampleCtx.fillRect(x, y, grid.cellW, grid.cellH);
      sampleCtx.strokeStyle = ink;
      sampleCtx.lineWidth = 8;
      sampleCtx.strokeRect(x + 4, y + 4, grid.cellW - 8, grid.cellH - 8);
      sampleCtx.fillStyle = ink;
      sampleCtx.fillText(`${row + 1}-${col + 1}`, x + grid.cellW / 2, y + grid.cellH / 2);
    }
  }
}
