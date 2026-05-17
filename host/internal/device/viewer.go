package device

// viewerHTML is the self-contained simulator front-end. It opens a Server-Sent
// Events stream of base64-encoded 1024-byte framebuffers and renders them at
// 30 FPS onto a Canvas styled like a real OLED screen. Two on-screen buttons
// (and keyboard 1/A and 2/B) POST presses to the host.
const viewerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Hackintosh — simulator</title>
<style>
  :root {
    --pixel-on:  #74dcff;
    --pixel-off: #07151b;
    --device-bg: #1a2a35;
    --device-border: #314757;
    --bg:        #0c1418;
    --fg:        #c9d4dc;
    --accent:    #74dcff;
  }
  * { box-sizing: border-box; }
  html, body {
    margin: 0; padding: 0;
    background: var(--bg);
    color: var(--fg);
    font-family: ui-monospace, "SF Mono", Menlo, Consolas, monospace;
    min-height: 100vh;
    display: flex; flex-direction: column; align-items: center; justify-content: center;
  }
  h1 {
    font-size: 14px; font-weight: 500; letter-spacing: 0.2em; text-transform: uppercase;
    color: var(--accent); margin: 0 0 18px 0;
    text-shadow: 0 0 12px rgba(116,220,255,.4);
  }
  .device {
    position: relative;
    width: 720px;
    background: var(--device-bg);
    border: 1px solid var(--device-border);
    border-radius: 16px;
    padding: 36px 36px 26px;
    box-shadow:
      0 24px 80px rgba(0,0,0,.5),
      inset 0 1px 0 rgba(255,255,255,.06);
  }
  .nameplate {
    position: absolute; top: 12px; left: 18px;
    font-size: 10px; letter-spacing: .25em; color: #6c8290; text-transform: uppercase;
  }
  .pill {
    position: absolute; top: 12px; right: 18px;
    font-size: 10px; letter-spacing: .15em;
    padding: 3px 8px; border: 1px solid var(--device-border); border-radius: 999px;
    background: rgba(0,0,0,.2);
    color: #8aa3b1;
  }
  .pill.live { color: #6aff8a; border-color: rgba(106,255,138,.4); }
  .oled-frame {
    background: #000;
    padding: 12px;
    border-radius: 8px;
    border: 1px solid #000;
    box-shadow: inset 0 0 24px rgba(0,0,0,.8), 0 1px 0 rgba(255,255,255,.04);
  }
  .oled {
    background: var(--pixel-off);
    border-radius: 4px;
    width: 100%; max-width: 640px; margin: 0 auto;
    aspect-ratio: 128 / 64;
    display: block;
  }
  canvas {
    width: 100%; height: 100%;
    image-rendering: pixelated;
    image-rendering: crisp-edges;
    display: block;
  }
  .buttons {
    display: flex; justify-content: center; gap: 80px; margin-top: 22px;
  }
  .btn {
    background: #0e1c25;
    color: var(--fg);
    border: 1px solid var(--device-border);
    border-radius: 12px;
    padding: 14px 26px;
    min-width: 180px;
    font: inherit; font-size: 12px; letter-spacing: .15em; text-transform: uppercase;
    cursor: pointer;
    user-select: none;
    transition: transform 30ms, background 100ms;
    box-shadow: 0 4px 0 #060c10, inset 0 1px 0 rgba(255,255,255,.04);
  }
  .btn:hover { background: #122430; }
  .btn:active, .btn.held { transform: translateY(2px); box-shadow: 0 2px 0 #060c10; background: #15303d; }
  .btn .key { display: block; color: #5d7b8a; font-size: 9px; margin-top: 4px; letter-spacing: .2em; }
  .help {
    margin-top: 22px;
    text-align: center;
    font-size: 11px; color: #5d7b8a; line-height: 1.6;
  }
  .help kbd {
    display: inline-block; padding: 1px 6px;
    border: 1px solid #314757; border-radius: 4px;
    background: rgba(0,0,0,.3); color: #8aa3b1;
    font-size: 10px;
  }
</style>
</head>
<body>
  <h1>Hackintosh simulator</h1>
  <div class="device">
    <span class="nameplate">XIAO RP2040 · SSD1306</span>
    <span id="status" class="pill">connecting…</span>
    <div class="oled-frame">
      <div class="oled">
        <canvas id="screen" width="128" height="64"></canvas>
      </div>
    </div>
    <div class="buttons">
      <button id="btnA" class="btn">A · Tea timer<span class="key">key A or 1</span></button>
      <button id="btnB" class="btn">B · Next screen<span class="key">key B or 2</span></button>
    </div>
  </div>
  <p class="help">
    Hold a button for <kbd>700ms</kbd> to send a long-press.
    Disconnect &mdash; just close this tab; reload to reconnect.
  </p>
<script>
(() => {
  const PIXEL_ON  = [116, 220, 255, 255];
  const PIXEL_OFF = [  7,  21,  27, 255];

  const cvs = document.getElementById('screen');
  const ctx = cvs.getContext('2d');
  ctx.imageSmoothingEnabled = false;
  const img = ctx.createImageData(128, 64);

  function render(frame) {
    // frame is a 1024-byte ArrayBuffer in SSD1306 page-major layout:
    //   offset = page*128 + col,  bit 0 = top pixel of the page.
    const data = img.data;
    for (let page = 0; page < 8; page++) {
      for (let col = 0; col < 128; col++) {
        const byte = frame[page * 128 + col];
        for (let bit = 0; bit < 8; bit++) {
          const y = page * 8 + bit;
          const x = col;
          const idx = (y * 128 + x) * 4;
          const on = (byte >> bit) & 1;
          const c = on ? PIXEL_ON : PIXEL_OFF;
          data[idx    ] = c[0];
          data[idx + 1] = c[1];
          data[idx + 2] = c[2];
          data[idx + 3] = c[3];
        }
      }
    }
    ctx.putImageData(img, 0, 0);
  }

  const status = document.getElementById('status');
  function setStatus(text, live) {
    status.textContent = text;
    status.classList.toggle('live', !!live);
  }

  function decodeBase64(b64) {
    const bin = atob(b64);
    const out = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
    return out;
  }

  function connect() {
    setStatus('connecting…', false);
    const es = new EventSource('/frames');
    es.onopen    = () => setStatus('live · 30fps', true);
    es.onerror   = () => { setStatus('disconnected — retrying', false); };
    es.onmessage = (e) => render(decodeBase64(e.data));
  }
  connect();

  // ---- Buttons ----
  function postButton(id, event) {
    fetch('/button', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id, event }),
    }).catch(() => {});
  }

  function wireButton(el, id) {
    let downAt = 0;
    let longTimer = 0;
    let sentLong = false;

    const down = (e) => {
      e.preventDefault();
      el.classList.add('held');
      downAt = performance.now();
      sentLong = false;
      postButton(id, 'press');
      longTimer = setTimeout(() => {
        sentLong = true;
        postButton(id, 'long');
      }, 700);
    };
    const up = (e) => {
      if (e) e.preventDefault();
      el.classList.remove('held');
      clearTimeout(longTimer);
      postButton(id, 'release');
    };
    el.addEventListener('mousedown', down);
    el.addEventListener('touchstart', down, { passive: false });
    el.addEventListener('mouseup', up);
    el.addEventListener('mouseleave', up);
    el.addEventListener('touchend', up);
    return { down, up };
  }
  const a = wireButton(document.getElementById('btnA'), 0);
  const b = wireButton(document.getElementById('btnB'), 1);

  // Keyboard: A/1 for A, B/2 for B
  const keyHeld = { 0: false, 1: false };
  window.addEventListener('keydown', (e) => {
    if (e.repeat) return;
    let id = -1;
    const k = e.key.toLowerCase();
    if (k === 'a' || k === '1') id = 0;
    else if (k === 'b' || k === '2') id = 1;
    if (id < 0) return;
    keyHeld[id] = true;
    (id === 0 ? a : b).down({ preventDefault() {} });
  });
  window.addEventListener('keyup', (e) => {
    let id = -1;
    const k = e.key.toLowerCase();
    if (k === 'a' || k === '1') id = 0;
    else if (k === 'b' || k === '2') id = 1;
    if (id < 0 || !keyHeld[id]) return;
    keyHeld[id] = false;
    (id === 0 ? a : b).up();
  });
})();
</script>
</body>
</html>`
