#!/usr/bin/env node
/**
 * extract.mjs — Extract static completion specs from @withfig/autocomplete.
 *
 * For each CLI in build/*.js, loads the spec, strips generators/dynamic
 * functions, and writes a clean JSON file to ../../sources/fig/raw/<name>.json.
 *
 * Usage:  node extract.mjs
 * Output: sources/fig/raw/<name>.json  (one per CLI)
 *
 * The output preserves the Fig v7 schema structure but only includes
 * static data: name, description, subcommands, options, args.
 */

import { createRequire } from "module";
import { readdirSync, writeFileSync, mkdirSync, existsSync } from "fs";
import { join, basename } from "path";

const require = createRequire(import.meta.url);
const BUILD_DIR = join(
  import.meta.dirname,
  "node_modules",
  "@withfig",
  "autocomplete",
  "build"
);
const OUT_DIR = join(import.meta.dirname, "..", "..", "sources", "fig", "raw");
const PROVIDER_DIR = join(import.meta.dirname, "..", "..", "engine", "provider");
const unconverted = [];
let currentCommand = "";

if (!existsSync(OUT_DIR)) {
  mkdirSync(OUT_DIR, { recursive: true });
}

// ── Helpers ──────────────────────────────────────────────────────────

/**
 * Strip non-serialisable fields (generators, functions, symbols)
 * from a Fig arg object.
 */
function cleanArg(arg, path = "args") {
  if (!arg || typeof arg !== "object") return arg;

  const out = {};
  for (const [k, v] of Object.entries(arg)) {
    if (v === undefined || v === null) continue;
    if (typeof v === "function") continue;
    if (typeof v === "object" && !Array.isArray(v)) {
      // Skip generators (they contain script/custom/postProcess functions)
      if (k === "generators") {
        recordUnconverted(v, path, "arg.generators");
        continue;
      }
      out[k] = cleanArg(v, `${path}.${k}`);
    } else if (Array.isArray(v)) {
      out[k] = v.map((item) => {
        if (typeof item === "function") return undefined;
        if (typeof item === "object" && item !== null) return cleanArg(item, path);
        return item;
      }).filter(Boolean);
    } else {
      out[k] = v;
    }
  }
  return out;
}

/**
 * Strip non-serialisable fields from a Fig option object.
 */
function cleanOption(opt, path = "option") {
  if (!opt || typeof opt !== "object") return opt;

  const out = {};
  for (const [k, v] of Object.entries(opt)) {
    if (v === undefined || v === null) continue;
    if (typeof v === "function") continue;
    if (typeof v === "object" && !Array.isArray(v)) {
      if (k === "generators") {
        recordUnconverted(v, path, "option.generators");
        continue;
      }
      out[k] = cleanArg(v, `${path}.${k}`);
    } else if (Array.isArray(v)) {
      // name can be string[]
      out[k] = v;
    } else {
      out[k] = v;
    }
  }
  return out;
}

/**
 * Recursively clean a completion spec tree.
 */
function cleanSpec(spec, path = currentCommand) {
  if (!spec || typeof spec !== "object") return spec;

  const out = {};
  for (const [k, v] of Object.entries(spec)) {
    if (v === undefined || v === null) continue;
    if (typeof v === "function") continue;
    if (typeof v === "symbol") continue;

    switch (k) {
      case "generators":
      case "generateSpec":
      case "additionalSuggestions":
        recordUnconverted(v, path, "generators");
        continue;

      case "subcommands":
        if (Array.isArray(v)) {
          out.subcommands = v.map((item) => cleanSpec(item, `${path} > ${item.name || "subcommand"}`));
        }
        break;

      case "options":
        if (Array.isArray(v)) {
          out.options = v.map((item) => cleanOption(item, `${path} > ${item.name || "option"}`));
        }
        break;

      case "args":
        if (Array.isArray(v)) {
          out.args = v.map((item) => cleanArg(item, `${path} > args`));
        } else if (v && typeof v === "object") {
          // Fig's `cd` uses a generator-only args object. Preserve the
          // semantic argument name after removing executable generators.
          if (spec.name === "cd" && v.generators) {
            out.args = [{ name: "dir" }];
          } else {
            const cleaned = cleanArg(v, `${path} > args`);
            if (cleaned && Object.keys(cleaned).length > 0) {
              out.args = [cleaned];
            }
          }
        }
        break;

      case "name":
        // Fig allows name: string | string[] — keep both forms
        out.name = v;
        break;

      default:
        // Keep other static fields (description, hidden, deprecated, etc.)
        if (typeof v === "object" && !Array.isArray(v)) {
          out[k] = cleanArg(v);
        } else {
          out[k] = v;
        }
    }
  }
  return out;
}

// ── Main ─────────────────────────────────────────────────────────────

const files = readdirSync(BUILD_DIR).filter(
  (f) => f.endsWith(".js") && !f.startsWith(".") && f !== "index.js"
);

console.log(`Found ${files.length} spec files in build/`);

let ok = 0;
let skipped = 0;
let errored = 0;

for (const file of files) {
  const name = basename(file, ".js");
  try {
    const mod = require(join(BUILD_DIR, file));
    const spec = mod.default || mod;

    if (!spec || !spec.name) {
      skipped++;
      continue;
    }

    currentCommand = Array.isArray(spec.name) ? spec.name[0] : spec.name;
    const cleaned = cleanSpec(spec, currentCommand);
    const json = JSON.stringify(cleaned, null, 2);
    writeFileSync(join(OUT_DIR, `${name}.json`), json + "\n");
    ok++;
  } catch (err) {
    // Some specs may have runtime errors when loaded — skip them
    console.error(`  SKIP ${name}: ${err.message}`);
    errored++;
  }
}

console.log(
  `Done: ${ok} extracted, ${skipped} skipped (no name), ${errored} errored`
);
console.log(`Output: ${OUT_DIR}/`);

mkdirSync(PROVIDER_DIR, { recursive: true });
const report = [
  "# Fig generators ainda não convertidos",
  "",
  "Relatório gerado automaticamente por `tools/fig/node/extract.mjs`.",
  "Nenhum generator listado abaixo deve ser descartado sem criar um placeholder/provider equivalente.",
  "",
  `Total de ocorrências não convertidas: ${unconverted.length}`,
  "",
];
for (const item of unconverted.sort((a, b) =>
  `${a.command}/${a.path}`.localeCompare(`${b.command}/${b.path}`))) {
  report.push(`- **Comando:** \`${item.command}\``);
  report.push(`  - **Caminho:** \`${item.path}\``);
  report.push(`  - **Generator:** \`${item.generator}\``);
  report.push(`  - **Campo:** \`${item.field}\``);
  report.push(`  - **Ação:** criar/mapeiar provider para este caso`);
  report.push("");
}
if (unconverted.length === 0) report.push("Nenhum generator pendente.", "");
writeFileSync(join(PROVIDER_DIR, "FIG_UNCONVERTED.md"), report.join("\n"));

function recordUnconverted(generator, path, field) {
  let label = "custom function";
  if (typeof generator === "string") label = generator;
  else if (generator && typeof generator === "object") {
    label = generator.template || generator.name ||
      (Object.keys(generator).length ? `custom object: ${Object.keys(generator).join(", ")}` : "custom object");
  }
  unconverted.push({ command: currentCommand || "unknown", path, field, generator: label });
}
