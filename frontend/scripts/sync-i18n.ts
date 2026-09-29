import fs from "fs";
import path from "path";

// Load API key strictly from environment or .env / .env.local (never hardcoded in code)
function getDeepLApiKey(): string {
  if (process.env.DEEPL_API_KEY) {
    return process.env.DEEPL_API_KEY.trim();
  }

  const candidatePaths = [
    path.resolve(import.meta.dir, "..", ".env.local"),
    path.resolve(import.meta.dir, "..", ".env"),
    path.resolve(import.meta.dir, "../..", ".env.local"),
    path.resolve(import.meta.dir, "../..", ".env"),
  ];

  for (const envPath of candidatePaths) {
    if (fs.existsSync(envPath)) {
      try {
        const lines = fs.readFileSync(envPath, "utf-8").split("\n");
        for (const line of lines) {
          const trimmed = line.trim();
          if (trimmed.startsWith("#") || !trimmed.includes("=")) continue;
          const [key, ...rest] = trimmed.split("=");
          if (key.trim() === "DEEPL_API_KEY") {
            return rest.join("=").trim().replace(/^["']|["']$/g, "");
          }
        }
      } catch {}
    }
  }
  return "";
}

const DEEPL_API_KEY = getDeepLApiKey();

function flattenObj(obj: any, prefix = ""): Record<string, string> {
  const result: Record<string, string> = {};
  for (const key of Object.keys(obj)) {
    const fullKey = prefix ? `${prefix}.${key}` : key;
    const val = obj[key];
    if (typeof val === "string") {
      result[fullKey] = val;
    } else if (typeof val === "object" && val !== null) {
      Object.assign(result, flattenObj(val, fullKey));
    }
  }
  return result;
}

function unflattenObj(flat: Record<string, string>): any {
  const result: any = {};
  for (const [key, val] of Object.entries(flat)) {
    const parts = key.split(".");
    let curr = result;
    for (let i = 0; i < parts.length - 1; i++) {
      const part = parts[i];
      if (!curr[part] || typeof curr[part] !== "object") {
        curr[part] = {};
      }
      curr = curr[part];
    }
    curr[parts[parts.length - 1]] = val;
  }
  return result;
}

// Mask placeholders like {name}, {count}, etc., using XML tags for DeepL
function maskPlaceholders(text: string): { masked: string; placeholders: string[] } {
  const placeholders: string[] = [];
  const masked = text.replace(/\{([a-zA-Z0-9_]+)\}/g, (match) => {
    placeholders.push(match);
    return `<x id="${placeholders.length - 1}"/>`;
  });
  return { masked, placeholders };
}

function unmaskPlaceholders(text: string, placeholders: string[]): string {
  let result = text;
  for (let i = 0; i < placeholders.length; i++) {
    // DeepL might format XML tags with or without spaces, e.g. <x id="0"/> or <x id="0" />
    const pattern = new RegExp(`<x id=["']?${i}["']?\\s*\\/?>`, "gi");
    result = result.replace(pattern, placeholders[i]);
  }
  return result;
}

async function translateBatch(
  apiKey: string,
  texts: string[],
  targetLang: string
): Promise<string[]> {
  const isFree = apiKey.endsWith(":fx");
  const host = isFree ? "api-free.deepl.com" : "api.deepl.com";
  const url = `https://${host}/v2/translate`;

  // DeepL target language for English is typically "EN-US" or "EN-GB", but "EN" also works in v2
  const target = targetLang.toUpperCase() === "EN" ? "EN-US" : targetLang.toUpperCase();

  const maskedItems = texts.map((t) => maskPlaceholders(t));
  const payloadTexts = maskedItems.map((m) => m.masked);

  const res = await fetch(url, {
    method: "POST",
    headers: {
      Authorization: `DeepL-Auth-Key ${apiKey}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      text: payloadTexts,
      source_lang: "TH",
      target_lang: target,
      tag_handling: "xml",
      ignore_tags: ["x"],
    }),
  });

  if (!res.ok) {
    const errText = await res.text();
    throw new Error(`DeepL API error (${res.status}): ${errText}`);
  }

  const data = await res.json();
  const translations: string[] = data.translations.map((item: any, idx: number) => {
    return unmaskPlaceholders(item.text, maskedItems[idx].placeholders);
  });

  return translations;
}

async function syncTarget(targetLang: string, thData: any, localesDir: string) {
  const targetFile = path.join(localesDir, `${targetLang.toLowerCase()}.json`);
  let existingData: any = {};
  if (fs.existsSync(targetFile)) {
    try {
      existingData = JSON.parse(fs.readFileSync(targetFile, "utf-8"));
    } catch {
      existingData = {};
    }
  }

  const flatTh = flattenObj(thData);
  const flatTarget = flattenObj(existingData);

  const missingKeys: string[] = [];
  const missingTexts: string[] = [];

  for (const [key, thText] of Object.entries(flatTh)) {
    if (!flatTarget[key] || flatTarget[key].trim() === "") {
      missingKeys.push(key);
      missingTexts.push(thText);
    }
  }

  console.log(`\n🌐 [ซิงค์ภาษา: ${targetLang.toUpperCase()}]`);
  console.log(`  • จำนวนคีย์ทั้งหมดใน Master Thai: ${Object.keys(flatTh).length}`);
  console.log(`  • จำนวนคีย์ที่แปลแล้วเดิม:       ${Object.keys(flatTarget).length - missingKeys.length}`);
  console.log(`  • จำนวนคีย์ที่ต้องส่งแปล DeepL:    ${missingKeys.length}`);

  if (missingKeys.length === 0) {
    console.log(`  \x1b[32m✔ แปลครบถ้วนแล้ว ไม่ต้องยิง API\x1b[0m`);
    return;
  }

  const totalChars = missingTexts.reduce((acc, t) => acc + t.length, 0);
  console.log(`  • ตัวอักษรรวมที่จะแปล:          ${totalChars.toLocaleString()} ตัวอักษร`);
  console.log(`  ⏳ กำลังส่งแปลไปยัง DeepL...`);

  // Batch translate in chunks of 50 to stay well within request size limits
  const BATCH_SIZE = 50;
  for (let i = 0; i < missingKeys.length; i += BATCH_SIZE) {
    const batchKeys = missingKeys.slice(i, i + BATCH_SIZE);
    const batchTexts = missingTexts.slice(i, i + BATCH_SIZE);

    const translatedBatch = await translateBatch(DEEPL_API_KEY, batchTexts, targetLang);
    for (let j = 0; j < batchKeys.length; j++) {
      flatTarget[batchKeys[j]] = translatedBatch[j];
    }
    process.stdout.write(`    ความคืบหน้า: ${Math.min(i + BATCH_SIZE, missingKeys.length)} / ${missingKeys.length} รายการ\r`);
  }

  console.log(`\n  \x1b[32m✔ แปลสำเร็จครบทุกรายการ!\x1b[0m`);

  // Reconstruct nested object preserving Master Thai key order
  const orderedResult: Record<string, string> = {};
  for (const key of Object.keys(flatTh)) {
    if (flatTarget[key]) {
      orderedResult[key] = flatTarget[key];
    }
  }

  const nested = unflattenObj(orderedResult);
  fs.writeFileSync(targetFile, JSON.stringify(nested, null, 2) + "\n", "utf-8");
  console.log(`  💾 บันทึกไฟล์เรียบร้อยที่: ${path.relative(process.cwd(), targetFile)}`);
}

async function main() {
  console.log("\n========================================================");
  console.log("   🚀 Lingo-Translate: DeepL Auto-Sync i18n Tool");
  console.log("========================================================\n");

  if (!DEEPL_API_KEY) {
    console.log("❌ ไม่พบคีย์ DEEPL_API_KEY ในระบบ");
    console.log("กรุณาระบุในไฟล์ .env.local หรือ Environment Variable");
    process.exit(1);
  }

  const frontendDir = path.resolve(import.meta.dir, "..");
  const localesDir = path.join(frontendDir, "src", "locales");
  const thFile = path.join(localesDir, "th.json");

  if (!fs.existsSync(thFile)) {
    console.log(`❌ ไม่พบไฟล์ Master Thai: ${thFile}`);
    process.exit(1);
  }

  const thData = JSON.parse(fs.readFileSync(thFile, "utf-8"));

  // Read target languages from CLI argument e.g. --targets=en or default to "en"
  const args = process.argv.slice(2);
  let targets = ["en"];
  for (const arg of args) {
    if (arg.startsWith("--targets=") || arg.startsWith("--target=")) {
      targets = arg.split("=")[1].split(",").map((s) => s.trim().toLowerCase());
    }
  }

  try {
    for (const target of targets) {
      await syncTarget(target, thData, localesDir);
    }
    console.log("\n========================================================");
    console.log("✨ ทำการซิงค์ภาษาด้วย DeepL เสร็จสิ้นอย่างสมบูรณ์!");
    console.log("========================================================\n");
  } catch (err: any) {
    console.error("\n❌ เกิดข้อผิดพลาดระหว่างแปล:", err?.message || err);
    process.exit(1);
  }
}

main();
