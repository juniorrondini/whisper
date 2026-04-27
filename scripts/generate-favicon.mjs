import { mkdirSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";

const size = 32;
const pixels = Buffer.alloc(size * size * 4);

function insideRoundRect(x, y, w, h, r) {
  const dx = Math.max(Math.abs(x - w / 2) - w / 2 + r, 0);
  const dy = Math.max(Math.abs(y - h / 2) - h / 2 + r, 0);
  return dx * dx + dy * dy <= r * r;
}

function setPixel(x, y, rgba) {
  if (x < 0 || y < 0 || x >= size || y >= size) return;
  const row = size - 1 - y;
  const offset = (row * size + x) * 4;
  pixels[offset] = rgba[2];
  pixels[offset + 1] = rgba[1];
  pixels[offset + 2] = rgba[0];
  pixels[offset + 3] = rgba[3];
}

function fill(predicate, rgbaOrFn) {
  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      if (predicate(x, y)) {
        setPixel(x, y, typeof rgbaOrFn === "function" ? rgbaOrFn(x, y) : rgbaOrFn);
      }
    }
  }
}

fill((x, y) => insideRoundRect(x, y, 32, 32, 7), [16, 24, 40, 255]);

const bubble = (x, y) => {
  const body = x >= 6 && x <= 26 && y >= 4 && y <= 22 && insideRoundRect(x - 6, y - 4, 20, 18, 5);
  const tail = x >= 10 && x <= 16 && y >= 20 && y <= 27 && y - 20 >= Math.abs(x - 12.5) * 1.2;
  return body || tail;
};

fill(bubble, (_x, y) => {
  const t = Math.max(0, Math.min(1, (y - 4) / 23));
  return [
    Math.round(56 * (1 - t) + 3 * t),
    Math.round(189 * (1 - t) + 105 * t),
    Math.round(248 * (1 - t) + 161 * t),
    255
  ];
});

function distanceToSegment(px, py, ax, ay, bx, by) {
  const dx = bx - ax;
  const dy = by - ay;
  const len = dx * dx + dy * dy;
  const t = Math.max(0, Math.min(1, ((px - ax) * dx + (py - ay) * dy) / len));
  const x = ax + t * dx;
  const y = ay + t * dy;
  return Math.hypot(px - x, py - y);
}

const wave = [
  [9, 11.4],
  [12.1, 18.5],
  [16, 10.8],
  [19.9, 18.5],
  [23, 11.4]
];

fill((x, y) => {
  for (let i = 0; i < wave.length - 1; i++) {
    if (distanceToSegment(x + 0.5, y + 0.5, wave[i][0], wave[i][1], wave[i + 1][0], wave[i + 1][1]) <= 1.65) {
      return true;
    }
  }
  return false;
}, [255, 255, 255, 255]);

const highlight = [
  [9.8, 11.4],
  [12.4, 17.4],
  [16, 10.1],
  [19.6, 17.4],
  [22.2, 11.4]
];

fill((x, y) => {
  for (let i = 0; i < highlight.length - 1; i++) {
    if (distanceToSegment(x + 0.5, y + 0.5, highlight[i][0], highlight[i][1], highlight[i + 1][0], highlight[i + 1][1]) <= 0.55) {
      return true;
    }
  }
  return false;
}, [186, 230, 253, 95]);

fill((x, y) => (x - 23.5) ** 2 + (y - 22.5) ** 2 <= 7, [16, 185, 129, 255]);
fill((x, y) => (x - 23.5) ** 2 + (y - 22.5) ** 2 <= 1.3, [255, 255, 255, 255]);

const andMask = Buffer.alloc(size * 4);
const dibSize = 40 + pixels.length + andMask.length;
const ico = Buffer.alloc(22 + dibSize);
ico.writeUInt16LE(0, 0);
ico.writeUInt16LE(1, 2);
ico.writeUInt16LE(1, 4);
ico[6] = size;
ico[7] = size;
ico[8] = 0;
ico[9] = 0;
ico.writeUInt16LE(1, 10);
ico.writeUInt16LE(32, 12);
ico.writeUInt32LE(dibSize, 14);
ico.writeUInt32LE(22, 18);
ico.writeUInt32LE(40, 22);
ico.writeInt32LE(size, 26);
ico.writeInt32LE(size * 2, 30);
ico.writeUInt16LE(1, 34);
ico.writeUInt16LE(32, 36);
ico.writeUInt32LE(0, 38);
ico.writeUInt32LE(pixels.length, 42);
pixels.copy(ico, 62);
andMask.copy(ico, 62 + pixels.length);

for (const path of ["assets/favicon.ico", "frontend/public/favicon.ico"]) {
  mkdirSync(dirname(path), { recursive: true });
  writeFileSync(path, ico);
}
