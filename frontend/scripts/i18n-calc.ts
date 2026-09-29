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

interface DeepLUsage {
  character_count: number;
  character_limit: number;
  start_time?: string;
  end_time?: string;
}

async function fetchDeepLUsage(apiKey: string): Promise<DeepLUsage | null> {
  const isFree = apiKey.endsWith(":fx");
  const host = isFree ? "api-free.deepl.com" : "api.deepl.com";
  const url = `https://${host}/v2/usage`;

  try {
    const res = await fetch(url, {
      headers: {
        Authorization: `DeepL-Auth-Key ${apiKey}`,
      },
    });

    if (!res.ok) {
      const errText = await res.text();
      console.error(`\x1b[31m[DeepL Error]\x1b[0m HTTP ${res.status}: ${errText}`);
      return null;
    }

    const data = await res.json();
    return {
      character_count: data.character_count ?? data.api_key_character_count ?? 0,
      character_limit: data.character_limit ?? data.api_key_character_limit ?? 0,
      start_time: data.start_time,
      end_time: data.end_time,
    };
  } catch (err: any) {
    console.error(`\x1b[31m[Network Error]\x1b[0m ไม่สามารถเชื่อมต่อกับ DeepL ได้:`, err?.message || err);
    return null;
  }
}

function countLocaleStrings(obj: any): { count: number; chars: number; entries: Record<string, string> } {
  let count = 0;
  let chars = 0;
  const entries: Record<string, string> = {};

  function traverse(current: any, prefix = "") {
    for (const key of Object.keys(current)) {
      const fullKey = prefix ? `${prefix}.${key}` : key;
      const val = current[key];
      if (typeof val === "string") {
        count++;
        chars += val.length;
        entries[fullKey] = val;
      } else if (typeof val === "object" && val !== null) {
        traverse(val, fullKey);
      }
    }
  }

  traverse(obj);
  return { count, chars, entries };
}

function scanSourceFiles(srcDir: string): { phrases: number; chars: number } {
  let phrases = 0;
  let chars = 0;

  function walk(dir: string) {
    if (!fs.existsSync(dir)) return;
    const items = fs.readdirSync(dir, { withFileTypes: true });
    for (const item of items) {
      const fullPath = path.join(dir, item.name);
      if (item.isDirectory()) {
        if (item.name !== "__tests__" && item.name !== "node_modules") {
          walk(fullPath);
        }
      } else if (item.isFile() && (item.name.endsWith(".tsx") || item.name.endsWith(".ts"))) {
        const content = fs.readFileSync(fullPath, "utf-8");
        // Extract JSX text
        const jsxMatches = content.matchAll(/>([^<>{}\n]+)</g);
        const cleanSet = new Set<string>();
        for (const m of jsxMatches) {
          const s = m[1].trim();
          if (s.length > 1 && !s.startsWith("http") && !/^[a-zA-Z0-9_-]+$/.test(s)) {
            cleanSet.add(s);
          }
        }
        // Extract quoted strings
        const strMatches = content.matchAll(/"([^"\n]{3,100})"/g);
        for (const m of strMatches) {
          const s = m[1].trim();
          if (s.length > 1 && !s.startsWith("http") && !s.includes("className") && s.includes(" ")) {
            cleanSet.add(s);
          }
        }
        for (const text of cleanSet) {
          phrases++;
          chars += text.length;
        }
      }
    }
  }

  walk(srcDir);
  return { phrases, chars };
}

async function main() {
  console.log("\n========================================================");
  console.log("   🌐 Lingo-Translate: เครื่องมือคำนวณโควตาการแปล DeepL");
  console.log("========================================================\n");

  if (!DEEPL_API_KEY) {
    console.log("❌ ไม่พบคีย์ DEEPL_API_KEY ในระบบ");
    console.log("กรุณาระบุคีย์ผ่าน Environment Variable หรือในไฟล์ .env.local (ซึ่งปลอดภัยและอยู่ใน .gitignore):");
    console.log("\n  ตัวอย่างสร้างไฟล์ .env.local:");
    console.log("    echo 'DEEPL_API_KEY=your_key_here' > .env.local\n");
    console.log("  หรือรันคำสั่งโดยตรง:");
    console.log("    DEEPL_API_KEY=your_key_here make i18n-calc\n");
    process.exit(1);
  }

  console.log("⏳ กำลังตรวจสอบสถานะบัญชี DeepL API...");
  const usage = await fetchDeepLUsage(DEEPL_API_KEY);

  if (!usage) {
    console.log("\n❌ ไม่สามารถดึงข้อมูลโควตาจาก DeepL ได้ กรุณาตรวจสอบความถูกต้องของ DEEPL_API_KEY");
    process.exit(1);
  }

  const remainingQuota = Math.max(0, usage.character_limit - usage.character_count);
  const usedPercent = usage.character_limit > 0 ? ((usage.character_count / usage.character_limit) * 100).toFixed(1) : "0";

  console.log("\n📊 [สถานะบัญชี DeepL ปัจจุบัน]");
  console.log(`  • โควตาทั้งหมด:     ${usage.character_limit.toLocaleString()} ตัวอักษร`);
  console.log(`  • ใช้งานไปแล้ว:     ${usage.character_count.toLocaleString()} ตัวอักษร (${usedPercent}%)`);
  console.log(`  • โควตาคงเหลือ:     \x1b[32m${remainingQuota.toLocaleString()}\x1b[0m ตัวอักษร`);
  if (usage.end_time) {
    const resetDate = new Date(usage.end_time).toLocaleDateString("th-TH", {
      year: "numeric",
      month: "long",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
    console.log(`  • รอบบิลจะรีเซ็ต:  ${resetDate}`);
  }

  // Scan UI text
  const frontendDir = path.resolve(import.meta.dir, "..");
  const localesDir = path.join(frontendDir, "src", "locales");
  const thFile = path.join(localesDir, "th.json");

  console.log("\n🔍 [การคำนวณข้อความ UI เดสท็อป]");

  let totalCharsToTranslate = 0;
  let totalPhrases = 0;

  if (fs.existsSync(thFile)) {
    try {
      const thData = JSON.parse(fs.readFileSync(thFile, "utf-8"));
      const thStats = countLocaleStrings(thData);
      totalPhrases = thStats.count;
      totalCharsToTranslate = thStats.chars;
      console.log(`  • พบไฟล์มาสเตอร์ภาษาไทย: src/locales/th.json`);
      console.log(`  • จำนวนคีย์/ข้อความ:     ${totalPhrases.toLocaleString()} ข้อความ`);
      console.log(`  • จำนวนตัวอักษรรวม:     ${totalCharsToTranslate.toLocaleString()} ตัวอักษร`);

      // Check missing keys in EN
      const enFile = path.join(localesDir, "en.json");
      if (fs.existsSync(enFile)) {
        try {
          const enData = JSON.parse(fs.readFileSync(enFile, "utf-8"));
          const enStats = countLocaleStrings(enData);
          let missingChars = 0;
          let missingCount = 0;
          for (const [key, text] of Object.entries(thStats.entries)) {
            if (!enStats.entries[key] || enStats.entries[key].trim() === "") {
              missingCount++;
              missingChars += text.length;
            }
          }
          console.log(`  • แปลเป็น EN แล้ว:        ${(thStats.count - missingCount).toLocaleString()} / ${thStats.count.toLocaleString()}`);
          if (missingCount > 0) {
            console.log(`  • คำที่ยังค้างต้องแปล (EN): \x1b[33m${missingCount.toLocaleString()} ข้อความ (${missingChars.toLocaleString()} ตัวอักษร)\x1b[0m`);
            totalCharsToTranslate = missingChars;
          } else {
            console.log(`  • สถานะภาษาอังกฤษ:       \x1b[32mแปลครบ 100% แล้ว ไม่ต้องใช้โควตาเพิ่ม\x1b[0m`);
          }
        } catch {}
      }
    } catch (err: any) {
      console.error("  ❌ ไม่สามารถอ่าน th.json ได้:", err.message);
    }
  } else {
    console.log(`  • ยังไม่พบ src/locales/th.json -> ประเมินจากโค้ด UI ใน src/ ทั้งหมด:`);
    const scanStats = scanSourceFiles(path.join(frontendDir, "src"));
    totalPhrases = scanStats.phrases;
    totalCharsToTranslate = scanStats.chars;
    console.log(`  • จำนวนข้อความใน UI:     ~${totalPhrases.toLocaleString()} วลี`);
    console.log(`  • ความยาวตัวอักษรโดยประมาณ: ~${totalCharsToTranslate.toLocaleString()} ตัวอักษร`);
  }

  // Cost / Feasibility estimation
  console.log("\n💡 [ผลการวิเคราะห์ความเพียงพอของโควตา]");
  const targets = [
    { code: "EN", name: "ภาษาอังกฤษ (English)", factor: 1 },
    { code: "EN + JA", name: "ภาษาอังกฤษ + ภาษาญี่ปุ่น (EN + JA)", factor: 2 },
    { code: "EN + JA + ZH", name: "อังกฤษ + ญี่ปุ่น + จีน (EN + JA + ZH)", factor: 3 },
  ];

  for (const t of targets) {
    const requiredChars = totalCharsToTranslate * t.factor;
    const reqPercent = remainingQuota > 0 ? ((requiredChars / remainingQuota) * 100).toFixed(2) : "0";
    const leftover = remainingQuota - requiredChars;
    const isSufficient = leftover >= 0;

    console.log(`\n  ▸ แผนการแปล: ${t.name}`);
    console.log(`    - ต้องใช้โควตา:     ${requiredChars.toLocaleString()} ตัวอักษร`);
    console.log(`    - สัดส่วนจากที่เหลือ:  ${reqPercent}% ของโควตาที่เหลือ`);
    if (isSufficient) {
      console.log(`    - โควตาคงเหลือหลังแปล: \x1b[32m${leftover.toLocaleString()} ตัวอักษร\x1b[0m`);
      console.log(`    - ผลลัพธ์:          \x1b[32m✔ โควตาเพียงพอแน่นอน (ปลอดภัยมาก)\x1b[0m`);
    } else {
      console.log(`    - ผลลัพธ์:          \x1b[31m✖ โควตาไม่เพียงพอ (ขาดอีก ${Math.abs(leftover).toLocaleString()} ตัวอักษร)\x1b[0m`);
    }
  }

  console.log("\n========================================================");
  console.log("📌 สรุป: โควตาของคุณเหลือเฟือสำหรับการแปล Desktop UI ครับ");
  console.log("========================================================\n");
}

main();
