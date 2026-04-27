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

function fill(predicate, rgba) {
  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      if (predicate(x, y)) setPixel(x, y, rgba);
    }
  }
}

fill((x, y) => insideRoundRect(x, y, 32, 32, 7), [16, 24, 40, 255]);
fill((x, y) => insideRoundRect(x - 4, y - 5, 24, 20, 6) && !(x > 21 && y > 20), [14, 165, 233, 255]);
fill((x, y) => x >= 11 && x <= 23 && y >= 12 && y <= 14, [255, 255, 255, 255]);
fill((x, y) => x >= 11 && x <= 19 && y >= 18 && y <= 20, [255, 255, 255, 255]);
fill((x, y) => (x - 24) ** 2 + (y - 24) ** 2 <= 20, [16, 185, 129, 255]);

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
