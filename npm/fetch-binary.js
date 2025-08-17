/**
 * @fileoverview Binary fetcher for md-social from GitHub releases
 * Node 18+
 * Env: OWNER, REPO, VERSION (or "latest"), ASSET (optional), GITHUB_TOKEN (optional)
 */
import { createWriteStream } from "node:fs";
import { chmod, mkdir, readFile } from "node:fs/promises";
import path from "node:path/win32";
import { Readable } from "node:stream";
import { pipeline } from "node:stream/promises";
import { promisify } from "node:util";
import zlib from "node:zlib";

/** @type {(source: NodeJS.ReadableStream, destination: NodeJS.WritableStream) => Promise<void>} */
const streamPipeline = promisify(pipeline);

/** @type {string} */
const base = `md-social-${getPlatform()}-${getArch()}`;

/** @type {{ version?: string }} */
const pkgJson = JSON.parse(await readFile("package.json", "utf8"));

/** @type {string} */
const toolname = "md-social";

/** @type {string} */
const OWNER = "andrioid";
/** @type {string} */
const REPO = "md-social";
/** @type {string} */
const VERSION = pkgJson.version || "latest";

// Try common filename variants. No predefined support list.
/** @type {string[]} */
const candidates = [
   base,
   `${base}.gz`,
   // Executable suffix on Windows is a common gotcha:
   process.platform === "win32" ? `${base}.exe.zip` : null,
   `${base}.zip`,
].filter((v) => v !== null);

/** @type {string} */
const prefix =
   VERSION === "latest" ? `https://github.com/${OWNER}/${REPO}/releases/latest/download/` : `https://github.com/${OWNER}/${REPO}/releases/download/${VERSION}/`;

const outDir = new URL("bin/", import.meta.url);
await mkdir(outDir, { recursive: true });
const outPath = new URL(`bin/${toolname}`, import.meta.url);

/** @type {Record<string, string>} */
const headers = process.env.GITHUB_TOKEN ? { Authorization: `Bearer ${process.env.GITHUB_TOKEN}` } : {};

/** @type {boolean} */
let downloaded = false;
for (const name of candidates) {
   downloaded = await downloadAndDecompress(name);
   if (downloaded) break;
}

if (!downloaded) {
   console.error(`Could not download any of: ${candidates.join(", ")} from ${prefix}`);
   // Let install fail (your requirement).
   process.exit(1);
}

/**
 * Maps Node.js process.arch to Go architecture names
 * @returns {string} The architecture string for the binary
 */
function getArch() {
   const arch = process.arch;
   switch (arch) {
      case "x64":
         return "amd64";
      case "arm64":
         return "arm64";
      case "arm":
         return "arm"; // may want to detect armv6 vs armv7
      case "ia32":
         return "386";
      case "ppc64":
         return "ppc64";
      case "s390x":
         return "s390x";
      case "riscv64":
         return "riscv64";
      default:
         return arch; // passthrough if unknown
   }
}

/**
 * Maps Node.js process.platform to Go platform names
 * @returns {string} The platform string for the binary
 */
function getPlatform() {
   const platform = process.platform;
   switch (platform) {
      case "darwin":
         return "darwin";
      case "linux":
         return "linux";
      case "win32":
         return "windows";
      default:
         return platform; // passthrough if unknown
   }
}

/**
 * Uncompress the file, using the appropriate tool
 * @param {string} ext The source file path
 */
function getUncompressor(ext) {
   // get file extension
   let handler = null;

   switch (ext) {
      case ".gz":
         handler = zlib.createGunzip();
      case ".zip":
         handler = zlib.createUnzip();
         break;
      default:
         throw new Error(`Unsupported compression format: ${ext}`);
   }

   return handler;
}

/**
 *
 * @param {string} name
 * @returns {Promise<boolean>} If we successfully downloaded
 */
async function downloadAndDecompress(name) {
   const url = prefix + encodeURIComponent(name);
   const res = await fetch(url, { headers });
   if (!res.ok) {
      // 404/403/etc.—try next candidate
      return false;
   }
   if (!res.body) return false;
   // Create a stream from res.body
   // @ts-ignore naughty types from node.js
   const srcStream = Readable.fromWeb(res.body);
   // extension from filename
   const ext = path.extname(name);
   const uncompressor = getUncompressor(ext);

   const dstPath = new URL(`bin/md-social`, import.meta.url);
   const dstStream = createWriteStream(dstPath);
   await pipeline(srcStream, uncompressor, dstStream);
   dstStream.close();

   // Setting permissions
   if (process.platform !== "win32") await chmod(outPath, 0o755);
   console.log(`Downloaded ${name} -> ${outPath.pathname}`);
   return true;
}
