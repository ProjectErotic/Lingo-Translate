import React, { useState, useMemo, useCallback } from "react";
import {
  Dialog,
  DialogContent,
  DialogTitle,
  Button,
  Badge,
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "@/ui";
import {
  BookOpen,
  ChevronLeft,
  ChevronRight,
  Sparkles,
  Cpu,
  ShieldCheck,
  Rocket,
  GitMerge,
  Share2,
  FolderOpen,
  FileCode,
  CheckCircle2,
  AlertTriangle,
  Lightbulb,
  Compass,
  Bookmark,
  Languages,
  Sliders,
  Database,
} from "lucide-react";

interface UserGuideDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

interface PageSpread {
  id: string;
  chapter: string;
  title: string;
  leftPage: {
    pageNumber: number;
    title: string;
    subtitle?: string;
    content: React.ReactNode;
  };
  rightPage: {
    pageNumber: number;
    title: string;
    subtitle?: string;
    content: React.ReactNode;
  };
}

export const UserGuideDialog: React.FC<UserGuideDialogProps> = ({
  open,
  onOpenChange,
}) => {
  const [currentSpreadIndex, setCurrentSpreadIndex] = useState(0);

  // Definition of spreads (2 pages per spread)
  const spreads: PageSpread[] = useMemo(
    () => [
      {
        id: "intro",
        chapter: "บทที่ 1: แนะนำโปรแกรม",
        title: "ภาพรวม & สถาปัตยกรรม Lingo Translate",
        leftPage: {
          pageNumber: 1,
          title: "ยินดีต้อนรับสู่ Lingo Translate",
          subtitle: "เครื่องมือแปลภาษาเกมสำหรับนักแปลและคอมมูนิตี้ไทย",
          content: (
            <div className="space-y-4 text-xs text-muted-foreground leading-relaxed">
              <p>
                <strong className="text-foreground">Lingo Translate</strong>{" "}
                คือโปรแกรมช่วยแปลภาษาเกมระดับมืออาชีพ พัฒนาด้วยเทคโนโลยี Go + Wails
                มีความเร็วสูง ปลอดภัย และออกแบบมาสำหรับคอมมูนิตี้ภาษาไทยโดยเฉพาะ
              </p>

              <div className="space-y-2 pt-1">
                <h4 className="font-semibold text-foreground flex items-center gap-1.5 text-xs">
                  <Sparkles className="w-3.5 h-3.5 text-primary" />
                  จุดเด่นที่สำคัญของโปรแกรม
                </h4>
                <div className="rounded-md border border-border bg-muted/40 p-2.5 space-y-2">
                  <div className="flex items-start gap-2">
                    <ShieldCheck className="w-4 h-4 text-emerald-400 shrink-0 mt-0.5" />
                    <div>
                      <span className="font-medium text-foreground block">
                        Non-Destructive Modding
                      </span>
                      ไม่ดัดแปลงหรือเขียนทับไฟล์เกมเดิม ใช้ระบบ Runtime Layer แทรกคำแปล
                      ป้องกันไฟล์เกมเสียหาย 100%
                    </div>
                  </div>
                  <div className="flex items-start gap-2">
                    <Cpu className="w-4 h-4 text-purple-400 shrink-0 mt-0.5" />
                    <div>
                      <span className="font-medium text-foreground block">
                        AI Multi-Provider
                      </span>
                      เชื่อมต่อ AI แปลภาษาได้หลากหลายค่าย (Gemini, GPT-4o, Claude, DeepL,
                      Ollama) พร้อมระบบแคชคำแปล
                    </div>
                  </div>
                  <div className="flex items-start gap-2">
                    <Database className="w-4 h-4 text-primary shrink-0 mt-0.5" />
                    <div>
                      <span className="font-medium text-foreground block">
                        SQLite High Performance
                      </span>
                      รองรับข้อความเกมมากกว่า 50,000+ ข้อความ ค้นหาและกรองได้ทันทีโดยไม่หน่วง
                    </div>
                  </div>
                </div>
              </div>

              <div className="rounded-md border border-border/80 bg-card p-2 text-[11px] text-muted-foreground flex items-center gap-2">
                <Compass className="w-4 h-4 text-primary shrink-0" />
                <span>
                  ใช้ปุ่มด้านล่างเพื่อพลิกหน้า หรือลากสไลเดอร์เพื่อข้ามบทได้อย่างรวดเร็ว
                </span>
              </div>
            </div>
          ),
        },
        rightPage: {
          pageNumber: 2,
          title: "เอนจินเกมที่รองรับ & ไฟล์งานแปล",
          subtitle: "รองรับโครงสร้างเกมสากลยอดนิยม",
          content: (
            <div className="space-y-4 text-xs text-muted-foreground leading-relaxed">
              <div className="space-y-2">
                <h4 className="font-semibold text-foreground text-xs">
                  เอนจินที่รองรับในปัจจุบัน
                </h4>
                <div className="grid grid-cols-2 gap-2">
                  <div className="p-2 rounded-md border border-border bg-muted/30">
                    <span className="font-semibold text-foreground block text-[11px]">
                      RPG Maker MV / MZ
                    </span>
                    <span className="text-[10px] text-muted-foreground">
                      ดึง Dialogue, Map Events, System, Database ครบถ้วน
                    </span>
                  </div>
                  <div className="p-2 rounded-md border border-border bg-muted/30">
                    <span className="font-semibold text-foreground block text-[11px]">
                      Ren'Py
                    </span>
                    <span className="text-[10px] text-muted-foreground">
                      สกัดสคริปต์ .rpy และ .rpyc เฉพาะบทสนทนาและช้อยส์
                    </span>
                  </div>
                  <div className="p-2 rounded-md border border-border bg-muted/30">
                    <span className="font-semibold text-foreground block text-[11px]">
                      Godot Engine
                    </span>
                    <span className="text-[10px] text-muted-foreground">
                      รองรับไฟล์ PO/CSV และ Text Localization
                    </span>
                  </div>
                  <div className="p-2 rounded-md border border-border bg-muted/30">
                    <span className="font-semibold text-foreground block text-[11px]">
                      Unity (TextAsset)
                    </span>
                    <span className="text-[10px] text-muted-foreground">
                      สกัดข้อความจาก TextAsset และชุดข้อมูลเกม
                    </span>
                  </div>
                </div>
              </div>

              <div className="space-y-2 pt-1">
                <h4 className="font-semibold text-foreground text-xs flex items-center gap-1.5">
                  <FileCode className="w-3.5 h-3.5 text-primary" />
                  ไฟล์โปรเจกต์ .nst (Workspace Archive)
                </h4>
                <p>
                  เมื่อสกัดเกม ข้อมูลทั้งหมดจะถูกรวบรวมไว้ในไฟล์แพ็กเกจเดี่ยว{" "}
                  <code className="px-1.5 py-0.5 rounded bg-muted text-primary font-mono text-[11px]">
                    .nst
                  </code>{" "}
                  (เช่น <code className="text-foreground">my_game.nst</code>) ภายในบรรจุฐานข้อมูล
                  SQLite และการตั้งค่า คุณสามารถก๊อบปี้ไฟล์นี้ไปแปลต่อในคอมพิวเตอร์เครื่องอื่นได้ทันที
                </p>
              </div>

              <div className="rounded-md border border-primary/20 bg-primary/5 p-2.5 flex items-start gap-2">
                <Lightbulb className="w-4 h-4 text-primary shrink-0 mt-0.5" />
                <span className="text-[11px]">
                  <strong>Pro Tip:</strong> คุณสามารถสำรองไฟล์ .nst ไว้เป็นเวอร์ชันย่อย เช่น{" "}
                  <code className="text-foreground">game_backup_ch1.nst</code> เพื่อความปลอดภัยของข้อมูล
                </span>
              </div>
            </div>
          ),
        },
      },
      {
        id: "extract",
        chapter: "บทที่ 2: การเปิดเกม",
        title: "การเปิดโฟลเดอร์เกม & สกัดข้อความ (Extract)",
        leftPage: {
          pageNumber: 3,
          title: "ขั้นตอนการเริ่มแปลเกมใหม่",
          subtitle: "ตรวจจับเอนจินอัตโนมัติและเตรียมโปรเจกต์",
          content: (
            <div className="space-y-4 text-xs text-muted-foreground leading-relaxed">
              <div className="space-y-2.5">
                <div className="flex items-start gap-2.5">
                  <div className="w-5 h-5 rounded-full bg-primary text-primary-foreground flex items-center justify-center font-bold text-[11px] shrink-0 mt-0.5">
                    1
                  </div>
                  <div>
                    <strong className="text-foreground block">คลิก Open Game Folder</strong>
                    กดปุ่ม <span className="text-foreground font-medium">Open Game Folder…</span>{" "}
                    ที่หน้าแรก หรือใช้คีย์ลัด <kbd className="px-1 py-0.5 bg-muted rounded border border-border text-[10px]">Ctrl+O</kbd>
                  </div>
                </div>

                <div className="flex items-start gap-2.5">
                  <div className="w-5 h-5 rounded-full bg-primary text-primary-foreground flex items-center justify-center font-bold text-[11px] shrink-0 mt-0.5">
                    2
                  </div>
                  <div>
                    <strong className="text-foreground block">เลือกโฟลเดอร์หลักของเกม</strong>
                    เลือกโฟลเดอร์ที่มีไฟล์รันเกม เช่น โฟลเดอร์ที่มีไฟล์ <code className="text-foreground">Game.exe</code> หรือมีโฟลเดอร์ <code className="text-foreground">www</code> / <code className="text-foreground">game</code>
                  </div>
                </div>

                <div className="flex items-start gap-2.5">
                  <div className="w-5 h-5 rounded-full bg-primary text-primary-foreground flex items-center justify-center font-bold text-[11px] shrink-0 mt-0.5">
                    3
                  </div>
                  <div>
                    <strong className="text-foreground block">ระบุภาษาและเริ่มสกัด</strong>
                    โปรแกรมจะตรวจจับ Engine ให้อัตโนมัติ เลือกระบุภาษาต้นฉบับ (เช่น Japanese หรือ English) และภาษาปลายทางเป็น Thai
                  </div>
                </div>
              </div>

              <div className="rounded-md border border-border bg-muted/40 p-2.5 space-y-1.5">
                <span className="font-semibold text-foreground block text-[11px]">
                  สิ่งที่เกิดขึ้นระหว่างการสกัด (Extraction)
                </span>
                <ul className="list-disc list-inside space-y-1 text-[11px]">
                  <li>แยกข้อความบทสนทนา (Dialogue) ออกจากโค้ดของเกม</li>
                  <li>คำนวณจำนวนคำและสร้างตารางฐานข้อมูล SQLite</li>
                  <li>สร้างระบบแคชสำหรับแปลและเตรียมพื้นที่สำหรับ Deploy</li>
                </ul>
              </div>
            </div>
          ),
        },
        rightPage: {
          pageNumber: 4,
          title: "ข้อแนะนำเฉพาะสำหรับแต่ละเอนจิน",
          subtitle: "เทคนิคการเตรียมไฟล์เกมก่อนเริ่มสกัด",
          content: (
            <div className="space-y-3.5 text-xs text-muted-foreground leading-relaxed">
              <div className="rounded-md border border-border p-2.5 space-y-1.5">
                <Badge variant="outline" className="text-[10px]">RPG Maker MV / MZ</Badge>
                <p className="text-[11px]">
                  หากเกมถูกแพ็คเป็นไฟล์เดี่ยว (Enigma Virtual Box หรือ C++ Wrapper) ให้แตกไฟล์เกมออกมาเป็นโฟลเดอร์ปกติที่มีโฟลเดอร์ <code className="text-foreground">www/data</code> เสียก่อน
                </p>
              </div>

              <div className="rounded-md border border-border p-2.5 space-y-1.5">
                <Badge variant="outline" className="text-[10px]">Ren'Py Engine</Badge>
                <p className="text-[11px]">
                  หากเกมมีเฉพาะไฟล์ <code className="text-foreground">.rpyc</code> (คอมไพล์แล้ว) ให้รันเกม 1 ครั้งเพื่อให้ตัวเกมคอมไพล์สคริปต์ หรือ Lingo จะทำการถอดรหัส bytecode อัตโนมัติ
                </p>
              </div>

              <div className="rounded-md border border-amber-500/20 bg-amber-500/5 p-2.5 flex items-start gap-2">
                <AlertTriangle className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
                <div className="text-[11px]">
                  <strong className="text-foreground block">ข้อควรระวัง:</strong>
                  หลีกเลี่ยงการย้ายโฟลเดอร์เกมต้นฉบับระหว่างที่กำลังเปิดแปลอยู่ในโปรแกรม หากต้องการย้าย ให้ปิดโปรเจกต์ก่อน
                </div>
              </div>

              <div className="flex items-center gap-2 pt-1 text-[11px]">
                <FolderOpen className="w-4 h-4 text-primary shrink-0" />
                <span>
                  หากต้องการเปิดงานเก่า ให้กด <kbd className="px-1 py-0.5 bg-muted rounded border border-border text-[10px]">Ctrl+Shift+O</kbd> เพื่อเลือกไฟล์ <code className="text-foreground">.nst</code>
                </span>
              </div>
            </div>
          ),
        },
      },
      {
        id: "settings-ai",
        chapter: "บทที่ 3: ตั้งค่า AI",
        title: "การเชื่อมต่อ AI Providers & API Keys",
        leftPage: {
          pageNumber: 5,
          title: "ผู้ให้บริการ AI ที่รองรับ",
          subtitle: "ตั้งค่า API Key ผ่านหน้าต่าง Settings (Ctrl+,)",
          content: (
            <div className="space-y-3.5 text-xs text-muted-foreground leading-relaxed">
              <p>
                เข้าสู่เมนู <strong className="text-foreground">Settings</strong>{" "}
                (<kbd className="px-1 py-0.5 bg-muted rounded border border-border text-[10px]">Ctrl+,</kbd>)
                เพื่อเชื่อมต่อ AI ผู้ช่วยแปลที่คุณต้องการ:
              </p>

              <div className="space-y-2">
                <div className="border border-border rounded-md p-2 bg-muted/20">
                  <div className="flex items-center justify-between">
                    <span className="font-semibold text-foreground text-[11px]">
                      1. Google Gemini (แนะนำสูงสุด)
                    </span>
                    <Badge variant="secondary" className="text-[10px]">แม่นยำ & ประหยัด</Badge>
                  </div>
                  <p className="text-[11px] mt-0.5">
                    เข้าใจภาษาไทยได้สละสลวย รองรับโมเดล Gemini 2.5 Flash / Pro มีความเร็วสูงและโควต้าฟรีจำนวนมาก
                  </p>
                </div>

                <div className="border border-border rounded-md p-2 bg-muted/20">
                  <div className="flex items-center justify-between">
                    <span className="font-semibold text-foreground text-[11px]">
                      2. OpenAI (GPT-4o / GPT-4o-mini)
                    </span>
                    <Badge variant="outline" className="text-[10px]">มาตรฐานสากล</Badge>
                  </div>
                  <p className="text-[11px] mt-0.5">
                    เหมาะสำหรับบทสนทนาที่ต้องการความเป๊ะตามหลักไวยากรณ์
                  </p>
                </div>

                <div className="border border-border rounded-md p-2 bg-muted/20">
                  <div className="flex items-center justify-between">
                    <span className="font-semibold text-foreground text-[11px]">
                      3. Anthropic Claude / DeepL / Ollama
                    </span>
                    <Badge variant="outline" className="text-[10px]">ทางเลือกยอดนิยม</Badge>
                  </div>
                  <p className="text-[11px] mt-0.5">
                    Ollama ช่วยให้คุณรันโมเดลภาษาในเครื่องตัวเองได้ 100% ไม่ต้องต่อเน็ต
                  </p>
                </div>
              </div>
            </div>
          ),
        },
        rightPage: {
          pageNumber: 6,
          title: "การปรับแต่งพารามิเตอร์การแปล",
          subtitle: "ควบคุมความสร้างสรรค์และทดสอบการเชื่อมต่อ",
          content: (
            <div className="space-y-3.5 text-xs text-muted-foreground leading-relaxed">
              <div className="space-y-2">
                <h4 className="font-semibold text-foreground text-xs flex items-center gap-1.5">
                  <Sliders className="w-3.5 h-3.5 text-primary" />
                  พารามิเตอร์สำคัญ
                </h4>
                <div className="rounded-md border border-border bg-muted/30 p-2.5 space-y-2 text-[11px]">
                  <div>
                    <span className="font-medium text-foreground block">Temperature (ความสร้างสรรค์)</span>
                    <span>แนะนำค่า 0.2 - 0.4 สำหรับเกม เพื่อรักษาความต่อเนื่องของชื่อเฉพาะและคำศัพท์</span>
                  </div>
                  <div>
                    <span className="font-medium text-foreground block">Max Concurrency & Rate Limit</span>
                    <span>กำหนดจำนวนการส่งคำขอพร้อมกัน เพื่อป้องกันการโดนบล็อกจาก API โควต้า</span>
                  </div>
                </div>
              </div>

              <div className="space-y-1.5 pt-1">
                <h4 className="font-semibold text-foreground text-xs flex items-center gap-1.5">
                  <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
                  ปุ่ม Test Connection
                </h4>
                <p className="text-[11px]">
                  เมื่อกรอก API Key แล้ว ให้กดปุ่ม <strong>Test Connection</strong> ในหน้าตั้งค่า
                  เพื่อตรวจสอบว่า Endpoint และ Key ใช้งานได้จริงก่อนเริ่มงานแปล
                </p>
              </div>

              <div className="rounded-md border border-primary/20 bg-primary/5 p-2 text-[11px]">
                💡 <strong>เคล็ดลับ:</strong> API Key จะถูกเข้ารหัสและบันทึกไว้ในเครื่องของคุณที่{" "}
                <code className="text-foreground">~/.config/lingo/settings.json</code> ไม่มีการส่งออกไปยังเซิร์ฟเวอร์ภายนอกใดๆ
              </div>
            </div>
          ),
        },
      },
      {
        id: "batch-ai",
        chapter: "บทที่ 4: การแปลด้วย AI",
        title: "ระบบแปลอัตโนมัติ (Batch Translation)",
        leftPage: {
          pageNumber: 7,
          title: "การสั่งแปลแบบชุด (Batch AI)",
          subtitle: "สั่งแปลหลายร้อยข้อความพร้อมกันในคลิกเดียว",
          content: (
            <div className="space-y-3.5 text-xs text-muted-foreground leading-relaxed">
              <p>
                เปิดหน้าต่างแปลอัตโนมัติด้วยปุ่ม <strong className="text-foreground">Batch Translate</strong>{" "}
                หรือกดคีย์ลัด <kbd className="px-1 py-0.5 bg-muted rounded border border-border text-[10px]">Ctrl+T</kbd>
              </p>

              <div className="space-y-2">
                <h4 className="font-semibold text-foreground text-xs">ตัวเลือกขอบเขตการแปล</h4>
                <div className="border border-border rounded-md p-2 bg-muted/20 space-y-1 text-[11px]">
                  <strong className="text-foreground block">Current File Only</strong>
                  <span>แปลเฉพาะไฟล์ที่กำลังเลือกอยู่ในหน้า Grid Editor เหมาะสำหรับทดสอบดูแนวทางการแปล</span>
                </div>
                <div className="border border-border rounded-md p-2 bg-muted/20 space-y-1 text-[11px]">
                  <strong className="text-foreground block">All Files in Project</strong>
                  <span>แปลทุกไฟล์ในเกมที่มีข้อความคงค้าง เหมาะสำหรับการแปลทั้งเกมแบบรวดเดียว</span>
                </div>
              </div>

              <div className="rounded-md border border-border bg-card p-2 text-[11px] space-y-1">
                <span className="font-semibold text-foreground block">Skip Translated Entries</span>
                <span>
                  ติ๊กตัวเลือกนี้เสมอเพื่อไม่ให้ AI แปลทับข้อความที่คุณเคยตรวจทานหรือแปลไปแล้ว ช่วยประหยัดค่า API
                </span>
              </div>
            </div>
          ),
        },
        rightPage: {
          pageNumber: 8,
          title: "Custom Prompt & Glossary (พจนานุกรม)",
          subtitle: "สั่ง AI ให้แปลได้ตรงใจและคุมโทนภาษาของเกม",
          content: (
            <div className="space-y-3.5 text-xs text-muted-foreground leading-relaxed">
              <div className="space-y-1.5">
                <h4 className="font-semibold text-foreground text-xs flex items-center gap-1.5">
                  <Languages className="w-3.5 h-3.5 text-primary" />
                  การระบุบริบทเกม (Context Prompt)
                </h4>
                <div className="rounded border border-border bg-muted/40 p-2 font-mono text-[10px] text-foreground">
                  "เกมแนวแฟนตาซียุคกลาง ตัวละครเป็นอัศวินหญิง สรรพนามพูดกับเจ้าหญิงใช้ 'ข้าพเจ้า' และ 'ท่านหญิง'"
                </div>
                <p className="text-[11px]">
                  ใส่บริบทในช่อง Prompt จะช่วยให้ AI เลือกใช้สรรพนามและสำนวนภาษาไทยได้สละสลวยขึ้นมาก
                </p>
              </div>

              <div className="space-y-1.5 pt-1">
                <h4 className="font-semibold text-foreground text-xs flex items-center gap-1.5">
                  <Bookmark className="w-3.5 h-3.5 text-amber-400" />
                  ระบบ Glossary ล็อกคำศัพท์
                </h4>
                <p className="text-[11px]">
                  คุณสามารถระบุคู่คำศัพท์ที่ห้ามแปลเพี้ยน เช่น:
                </p>
                <div className="rounded border border-border bg-muted/30 p-2 text-[10px] space-y-0.5">
                  <div><code className="text-primary">Eldoria</code> ➔ <code className="text-foreground">เอลโดเรีย</code> (ชื่อเมือง)</div>
                  <div><code className="text-primary">Healing Herb</code> ➔ <code className="text-foreground">สมุนไพรรักษา</code> (ไอเทม)</div>
                  <div><code className="text-primary">Fireball</code> ➔ <code className="text-foreground">ลูกไฟเวทมนตร์</code> (สกิล)</div>
                </div>
              </div>
            </div>
          ),
        },
      },
      {
        id: "editor-grid",
        chapter: "บทที่ 5: หน้าต่างแปล Grid",
        title: "การใช้งาน Translation Grid Editor",
        leftPage: {
          pageNumber: 9,
          title: "ส่วนประกอบของหน้าต่าง Editor",
          subtitle: "ตารางแก้ไขคำแปลความเร็วสูงรองรับ 50,000+ ข้อความ",
          content: (
            <div className="space-y-3 text-xs text-muted-foreground leading-relaxed">
              <div className="space-y-2">
                <div className="border-l-2 border-primary pl-2.5 space-y-0.5">
                  <strong className="text-foreground block text-[11px]">
                    1. แผงรายการไฟล์ (Left Sidebar)
                  </strong>
                  <span className="text-[11px]">
                    แสดงรายชื่อไฟล์ทั้งหมดของเกม สามารถลากปรับขยายความกว้างได้ พร้อมแถบสีบอกความคืบหน้า (%)
                  </span>
                </div>

                <div className="border-l-2 border-emerald-400 pl-2.5 space-y-0.5">
                  <strong className="text-foreground block text-[11px]">
                    2. แถบค้นหา & ตัวกรอง (Search & Filter)
                  </strong>
                  <span className="text-[11px]">
                    กด <kbd className="px-1 py-0.5 bg-muted rounded border border-border text-[10px]">Ctrl+F</kbd> เพื่อค้นหาคำแบบ Realtime ทั้งภาษาเดิมและคำแปล พร้อมปุ่มกรอง (ทั้งหมด / ยังไม่แปล / แปลแล้ว)
                  </span>
                </div>

                <div className="border-l-2 border-purple-400 pl-2.5 space-y-0.5">
                  <strong className="text-foreground block text-[11px]">
                    3. แผงรายละเอียดด้านล่าง (Bottom Detail Panel)
                  </strong>
                  <span className="text-[11px]">
                    แสดงข้อความต้นฉบับเต็ม บันทึกช่วยจำ และตัวอย่างเปรียบเทียบ สามารถลากปรับความสูงได้
                  </span>
                </div>
              </div>
            </div>
          ),
        },
        rightPage: {
          pageNumber: 10,
          title: "การแก้ไขคำแปล & รหัสควบคุม (Tokens)",
          subtitle: "ความปลอดภัยในการรักษาโค้ดของเกม",
          content: (
            <div className="space-y-3 text-xs text-muted-foreground leading-relaxed">
              <div className="space-y-1.5">
                <h4 className="font-semibold text-foreground text-xs">
                  การพิมพ์แก้ไขคำแปล (Inline Edit)
                </h4>
                <p className="text-[11px]">
                  ดับเบิ้ลคลิกที่ช่องคำแปล หรือกด <kbd className="px-1 py-0.5 bg-muted rounded border border-border text-[10px]">Enter</kbd> เพื่อเริ่มพิมพ์ เมื่อพิมพ์เสร็จให้กด <kbd className="px-1 py-0.5 bg-muted rounded border border-border text-[10px]">Enter</kbd> อีกครั้งเพื่อบันทึกและเลื่อนไปแถวถัดไป
                </p>
              </div>

              <div className="rounded-md border border-amber-500/20 bg-amber-500/5 p-2.5 space-y-1.5">
                <div className="flex items-center gap-1.5 text-amber-400 font-semibold text-xs">
                  <AlertTriangle className="w-3.5 h-3.5" />
                  รหัสควบคุมและตัวแปร (Escape Tokens)
                </div>
                <p className="text-[11px]">
                  ในข้อความเกมมักจะมีแท็กพิเศษ เช่น <code className="text-foreground">\V[1]</code>,{" "}
                  <code className="text-foreground">\N[2]</code>, <code className="text-foreground">\C[0]</code>,{" "}
                  หรือ <code className="text-foreground">{`{color}`}</code>
                </p>
                <p className="text-[11px] text-amber-200/90 dark:text-amber-300/80">
                  ⚠️ <strong>ห้ามลบหรือแปลแท็กเหล่านี้:</strong> เป็นตัวแปรที่เกมใช้ดึงชื่อตัวละครหรือเปลี่ยนสีตัวอักษร หากลบออกอาจทำให้เกม Error ได้
                </p>
              </div>
            </div>
          ),
        },
      },
      {
        id: "deploy",
        chapter: "บทที่ 6: การ Deploy ม็อด",
        title: "การติดตั้งม็อดเข้าเกม (Non-Destructive)",
        leftPage: {
          pageNumber: 11,
          title: "ระบบ Non-Destructive Modding",
          subtitle: "ติดตั้งม็อดภาษาไทยโดยไฟล์เดิมไม่เสียหาย",
          content: (
            <div className="space-y-3.5 text-xs text-muted-foreground leading-relaxed">
              <p>
                เมื่อแปลเสร็จแล้ว ให้กดปุ่ม <strong className="text-foreground">Deploy to Game…</strong>{" "}
                หรือใช้คีย์ลัด <kbd className="px-1 py-0.5 bg-muted rounded border border-border text-[10px]">Ctrl+D</kbd>
              </p>

              <div className="space-y-2">
                <h4 className="font-semibold text-foreground text-xs flex items-center gap-1.5">
                  <ShieldCheck className="w-3.5 h-3.5 text-emerald-400" />
                  เลเยอร์ทำงานอย่างไร?
                </h4>
                <div className="border border-border rounded-md p-2.5 bg-muted/30 space-y-1.5 text-[11px]">
                  <p>
                    Lingo จะไม่เขียนทับไฟล์ <code className="text-foreground">Map001.json</code> หรือสคริปต์ของเกม แต่จะวางไฟล์เลเยอร์รันไทม์ เช่น:
                  </p>
                  <ul className="list-disc list-inside space-y-1 text-foreground font-mono text-[10px]">
                    <li>js/plugins/Lingo_TranslationLayer.js</li>
                    <li>game/tl/thai/00_lingo_layer.rpy</li>
                  </ul>
                  <p>
                    เมื่อผู้เล่นเปิดเกม สคริปต์นี้จะสลับข้อความเป็นภาษาไทยในหน่วยความจำทันที
                  </p>
                </div>
              </div>
            </div>
          ),
        },
        rightPage: {
          pageNumber: 12,
          title: "การฝังฟอนต์ไทย & ทดสอบเปิดเกม",
          subtitle: "แก้ปัญหาสระลอยและตัดคำสวยงาม",
          content: (
            <div className="space-y-3.5 text-xs text-muted-foreground leading-relaxed">
              <div className="space-y-1.5">
                <h4 className="font-semibold text-foreground text-xs flex items-center gap-1.5">
                  <Sparkles className="w-3.5 h-3.5 text-primary" />
                  ฟอนต์ภาษาไทยมาตรฐาน
                </h4>
                <p className="text-[11px]">
                  ระบบ Deploy จะติดตั้งชุดฟอนต์ภาษาไทยที่ผ่านการปรับจูนช่องไฟและตัดสระลอย ทำให้ข้อความในกล่องสนทนาอ่านง่าย ไม่ซ้อนทับกัน
                </p>
              </div>

              <div className="space-y-1.5 pt-1">
                <h4 className="font-semibold text-foreground text-xs flex items-center gap-1.5">
                  <Rocket className="w-3.5 h-3.5 text-amber-400" />
                  การทดสอบเปิดเกมจริง
                </h4>
                <p className="text-[11px]">
                  หลังกด Deploy เรียบร้อย ให้ไปที่โฟลเดอร์เกมแล้วดับเบิ้ลคลิกเปิดไฟล์ <code className="text-foreground">Game.exe</code> เพื่อทดสอบบทสนทนาและเมนูภาษาไทยได้ทันที
                </p>
              </div>

              <div className="rounded-md border border-border bg-card p-2 text-[11px]">
                🛡️ <strong>การถอนการติดตั้ง:</strong> หากต้องการนำม็อดออกเพื่อกลับเป็นภาษาดั้งเดิม สามารถกดปุ่ม Uninstall หรือลบไฟล์ปลั๊กอินออก ตัวเกมจะคืนสภาพสมบูรณ์เหมือนเดิม 100%
              </div>
            </div>
          ),
        },
      },
      {
        id: "smart-merge",
        chapter: "บทที่ 7: อัปเดตเวอร์ชันเกม",
        title: "Smart Merge: อัปเดตแพทช์เกมใหม่อัตโนมัติ",
        leftPage: {
          pageNumber: 13,
          title: "แก้ปัญหาเมื่อผู้พัฒนาเกมปล่อยแพทช์ใหม่",
          subtitle: "ไม่ต้องเริ่มต้นแปลใหม่ตั้งแต่แรกเมื่อเกมอัปเดต",
          content: (
            <div className="space-y-3.5 text-xs text-muted-foreground leading-relaxed">
              <p>
                ปัญหาใหญ่ที่สุดของนักแปลเกม คือเมื่อแปลเกมเวอร์ชัน 1.0 ไปแล้ว ผู้สร้างเกมปล่อยเวอร์ชัน 1.1 ออกมา ทำให้ต้องเสียเวลาแปลใหม่อีกรอบ
              </p>

              <div className="rounded-md border border-border bg-muted/40 p-2.5 space-y-2">
                <div className="flex items-start gap-2">
                  <GitMerge className="w-4 h-4 text-emerald-400 shrink-0 mt-0.5" />
                  <div>
                    <strong className="text-foreground block text-[11px]">
                      ระบบ Smart Merge อัจฉริยะ
                    </strong>
                    <span className="text-[11px]">
                      โปรแกรมจะวิเคราะห์ความเหมือนของข้อความเดิมและบริบทแวดล้อม จากนั้นดึงคำแปลที่คุณเคยทำไว้มาใส่ในเกมเวอร์ชันใหม่ให้อัตโนมัติ
                    </span>
                  </div>
                </div>
              </div>

              <p className="text-[11px]">
                สามารถเปิดใช้ฟังก์ชันนี้ได้ผ่านเมนู <strong className="text-foreground">Deploy → Smart Merge</strong> หรือคีย์ลัด <kbd className="px-1 py-0.5 bg-muted rounded border border-border text-[10px]">Ctrl+M</kbd>
              </p>
            </div>
          ),
        },
        rightPage: {
          pageNumber: 14,
          title: "ขั้นตอนการ Merge ทีละสเต็ป",
          subtitle: "จับคู่ข้อความเก่าเข้าเวอร์ชันใหม่อย่างแม่นยำ",
          content: (
            <div className="space-y-3 text-xs text-muted-foreground leading-relaxed">
              <div className="space-y-2">
                <div className="flex items-start gap-2">
                  <Badge variant="outline" className="text-[10px] shrink-0">สเต็ป 1</Badge>
                  <span className="text-[11px]">สกัดโฟลเดอร์เกมเวอร์ชันใหม่ให้เรียบร้อย</span>
                </div>
                <div className="flex items-start gap-2">
                  <Badge variant="outline" className="text-[10px] shrink-0">สเต็ป 2</Badge>
                  <span className="text-[11px]">เปิดหน้าต่าง Smart Merge (<kbd className="px-1 py-0.5 bg-muted rounded text-[10px]">Ctrl+M</kbd>)</span>
                </div>
                <div className="flex items-start gap-2">
                  <Badge variant="outline" className="text-[10px] shrink-0">สเต็ป 3</Badge>
                  <span className="text-[11px]">เลือกไฟล์โปรเจกต์เดิม (<code className="text-foreground">.nst</code>) ที่เคยแปลไว้</span>
                </div>
                <div className="flex items-start gap-2">
                  <Badge variant="outline" className="text-[10px] shrink-0">สเต็ป 4</Badge>
                  <span className="text-[11px]">กด Start Merge เพื่อเริ่มการจับคู่คำแปล</span>
                </div>
              </div>

              <div className="rounded-md border border-emerald-500/20 bg-emerald-500/5 p-2.5 text-[11px]">
                🎉 <strong>ผลลัพธ์:</strong> ข้อความเดิมจะกลายเป็นสถานะ Translated ทันที คุณจะมีหน้าที่แค่แปลเฉพาะข้อความใหม่ที่เพิ่มเข้ามาเท่านั้น!
              </div>
            </div>
          ),
        },
      },
      {
        id: "publish",
        chapter: "บทที่ 8: การเผยแพร่ผลงาน",
        title: "การ Export & เผยแพร่ม็อดสู่ Chanomhub",
        leftPage: {
          pageNumber: 15,
          title: "การจัดทำแพ็กเกจม็อดสำเร็จรูป",
          subtitle: "ส่งมอบผลงานแปลให้ผู้เล่นนำไปติดตั้งได้ง่ายๆ",
          content: (
            <div className="space-y-3.5 text-xs text-muted-foreground leading-relaxed">
              <p>
                เปิดหน้าต่างเผยแพร่ม็อดผ่านเมนู <strong className="text-foreground">Deploy → Publish Mod to Chanomhub…</strong>{" "}
                หรือคีย์ลัด <kbd className="px-1 py-0.5 bg-muted rounded border border-border text-[10px]">Ctrl+P</kbd>
              </p>

              <div className="space-y-2">
                <h4 className="font-semibold text-foreground text-xs">ข้อมูลที่ต้องระบุ</h4>
                <div className="border border-border rounded-md p-2 bg-muted/20 space-y-1.5 text-[11px]">
                  <div>
                    <span className="font-medium text-foreground block">ชื่อม็อด & เครดิตผู้แปล</span>
                    <span>ใส่ชื่อม็อดภาษาไทย และชื่อทีมหรือนามแฝงของคุณเพื่อเป็นเกียรติในผลงาน</span>
                  </div>
                  <div>
                    <span className="font-medium text-foreground block">เวอร์ชันของม็อด & คำแนะนำ</span>
                    <span>ระบุเวอร์ชัน เช่น v1.0 พร้อมคำอธิบายวิธีการนำไปลงเกม</span>
                  </div>
                </div>
              </div>
            </div>
          ),
        },
        rightPage: {
          pageNumber: 16,
          title: "การแจกจ่าย & Chanomhub Integration",
          subtitle: "แพลตฟอร์มศูนย์รวมม็อดเกมภาษาไทย",
          content: (
            <div className="space-y-3.5 text-xs text-muted-foreground leading-relaxed">
              <div className="space-y-1.5">
                <h4 className="font-semibold text-foreground text-xs flex items-center gap-1.5">
                  <Share2 className="w-3.5 h-3.5 text-primary" />
                  การสร้างไฟล์ Zip / Mod Package
                </h4>
                <p className="text-[11px]">
                  โปรแกรมจะทำการรวมไฟล์เลเยอร์คำแปลและฟอนต์เป็นไฟล์ Zip ให้โดยอัตโนมัติ ผู้เล่นอื่นสามารถดาวน์โหลดไปแตกไฟล์ทับลงโฟลเดอร์เกมเพื่อเล่นได้ทันที
                </p>
              </div>

              <div className="rounded-md border border-border bg-card p-2.5 space-y-1 text-[11px]">
                <strong className="text-foreground block">การแชร์ลงคอมมูนิตี้</strong>
                <span>
                  คุณสามารถนำไฟล์ม็อดไปโพสต์ลงกลุ่มเกม หรืออัปโหลดเข้าสู่แพลตฟอร์ม Chanomhub เพื่อให้ผู้เล่นค้นหาและดาวน์โหลดผ่าน Mod Manager ได้อย่างสะดวก
                </span>
              </div>
            </div>
          ),
        },
      },
      {
        id: "shortcuts",
        chapter: "บทที่ 9: คีย์ลัด & เคล็ดลับ",
        title: "คีย์ลัดทั้งหมดในโปรแกรม & Pro Tips",
        leftPage: {
          pageNumber: 17,
          title: "ตารางคีย์ลัดสากล (Cheat Sheet)",
          subtitle: "เพิ่มความรวดเร็วในการทำงานด้วยคีย์บอร์ด",
          content: (
            <div className="space-y-2 text-xs text-muted-foreground leading-relaxed">
              <div className="rounded-md border border-border overflow-hidden text-[11px]">
                <table className="w-full text-left">
                  <tbody className="divide-y divide-border">
                    <tr className="bg-muted/40">
                      <td className="p-1.5 font-mono text-primary">Ctrl + O</td>
                      <td className="p-1.5 text-foreground">เปิดโฟลเดอร์เกม (Open Game)</td>
                    </tr>
                    <tr>
                      <td className="p-1.5 font-mono text-primary">Ctrl + Shift + O</td>
                      <td className="p-1.5 text-foreground">เปิดไฟล์งานแปล (.nst)</td>
                    </tr>
                    <tr className="bg-muted/40">
                      <td className="p-1.5 font-mono text-primary">Ctrl + W</td>
                      <td className="p-1.5 text-foreground">ปิดโปรเจกต์ปัจจุบัน</td>
                    </tr>
                    <tr>
                      <td className="p-1.5 font-mono text-primary">Ctrl + T</td>
                      <td className="p-1.5 text-foreground">เปิดหน้าต่าง AI Translate</td>
                    </tr>
                    <tr className="bg-muted/40">
                      <td className="p-1.5 font-mono text-primary">Ctrl + D</td>
                      <td className="p-1.5 text-foreground">ติดตั้งม็อดเข้าเกม (Deploy)</td>
                    </tr>
                    <tr>
                      <td className="p-1.5 font-mono text-primary">Ctrl + M</td>
                      <td className="p-1.5 text-foreground">อัปเดตเกมใหม่ (Smart Merge)</td>
                    </tr>
                    <tr className="bg-muted/40">
                      <td className="p-1.5 font-mono text-primary">Ctrl + P</td>
                      <td className="p-1.5 text-foreground">เผยแพร่ม็อด (Publish)</td>
                    </tr>
                    <tr>
                      <td className="p-1.5 font-mono text-primary">Ctrl + ,</td>
                      <td className="p-1.5 text-foreground">เปิดการตั้งค่า (Settings)</td>
                    </tr>
                    <tr className="bg-muted/40">
                      <td className="p-1.5 font-mono text-primary">F1</td>
                      <td className="p-1.5 text-foreground">เปิดคู่มือเล่มนี้ (User Guide)</td>
                    </tr>
                    <tr>
                      <td className="p-1.5 font-mono text-primary">Ctrl + F</td>
                      <td className="p-1.5 text-foreground">ค้นหาในตารางคำแปล</td>
                    </tr>
                    <tr className="bg-muted/40">
                      <td className="p-1.5 font-mono text-primary">Enter</td>
                      <td className="p-1.5 text-foreground">บันทึก & เลื่อนไปแถวถัดไป</td>
                    </tr>
                    <tr>
                      <td className="p-1.5 font-mono text-primary">← / →</td>
                      <td className="p-1.5 text-foreground">พลิกหน้าคู่มือ ถอยหลัง / ไปหน้า</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          ),
        },
        rightPage: {
          pageNumber: 18,
          title: "ข้อแนะนำระดับเซียน (Pro Tips)",
          subtitle: "คำแนะนำเพื่อการแปลเกมอย่างราบรื่น",
          content: (
            <div className="space-y-3.5 text-xs text-muted-foreground leading-relaxed">
              <div className="space-y-2">
                <div className="border border-border rounded-md p-2 bg-muted/20 space-y-1">
                  <strong className="text-foreground block text-[11px]">
                    1. แบ่งแปลทีละบท / ทีละแผนที่
                  </strong>
                  <p className="text-[11px]">
                    ในหน้า Editor ให้คลิกเลือกไฟล์เฉพาะ Map ที่ผู้เล่นต้องเจอช่วงต้นเกม แล้วกดทดสอบ Deploy เล่นจริง จะช่วยให้เห็นสไตล์คำแปลและปรับแก้ได้เร็วกว่ารอแปลจบทั้งเกม
                  </p>
                </div>

                <div className="border border-border rounded-md p-2 bg-muted/20 space-y-1">
                  <strong className="text-foreground block text-[11px]">
                    2. การปรับเปลี่ยนธีมและขนาดตัวอักษร
                  </strong>
                  <p className="text-[11px]">
                    หากรู้สึกว่าตัวหนังสือในตารางเล็กหรือใหญ่เกินไป สามารถเข้าไปที่{" "}
                    <code className="text-foreground">Settings → Appearance</code> เพื่อปรับแต่ง Font Size และ Density ตามที่สบายตาที่สุด
                  </p>
                </div>

                <div className="border border-border rounded-md p-2 bg-muted/20 space-y-1">
                  <strong className="text-foreground block text-[11px]">
                    3. ช่องทางรายงานปัญหา & สนับสนุน
                  </strong>
                  <p className="text-[11px]">
                    หากพบเกมที่ไม่สามารถสกัดได้ หรือเจอบั๊กระหว่างใช้งาน สามารถเปิด Issue ได้ที่ GitHub Repository:{" "}
                    <span className="text-primary font-mono text-[10px]">ProjectErotic/Lingo-Translate</span>
                  </p>
                </div>
              </div>
            </div>
          ),
        },
      },
    ],
    []
  );

  const currentSpread = spreads[currentSpreadIndex];
  const totalSpreads = spreads.length;
  const totalPages = totalSpreads * 2;

  const handlePrev = useCallback(() => {
    setCurrentSpreadIndex((prev) => Math.max(0, prev - 1));
  }, []);

  const handleNext = useCallback(() => {
    setCurrentSpreadIndex((prev) => Math.min(totalSpreads - 1, prev + 1));
  }, [totalSpreads]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "ArrowLeft") {
      e.preventDefault();
      handlePrev();
    } else if (e.key === "ArrowRight") {
      e.preventDefault();
      handleNext();
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-w-5xl w-full h-[88vh] p-0 flex flex-col overflow-hidden gap-0 bg-card border-border shadow-2xl focus:outline-none"
        onKeyDown={handleKeyDown}
        tabIndex={0}
      >
        {/* Top Header Bar */}
        <div className="h-12 border-b border-border bg-popover/80 px-4 flex items-center justify-between shrink-0 select-none">
          <div className="flex items-center gap-2">
            <BookOpen className="w-4 h-4 text-primary" />
            <DialogTitle className="text-sm font-semibold text-foreground">
              คู่มือการใช้งาน Lingo Translate
            </DialogTitle>
            <Badge variant="outline" className="text-[10px] hidden sm:inline-flex">
              {currentSpread.chapter}
            </Badge>
          </div>

          <div className="flex items-center gap-2 pr-6">
            {/* Chapter Selector Dropdown */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" className="h-7 text-xs gap-1.5 font-normal">
                  <Bookmark className="w-3.5 h-3.5 text-primary" />
                  <span className="hidden sm:inline">เลือกบท (สารบัญ)</span>
                  <span className="sm:hidden">สารบัญ</span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-64 max-h-72 overflow-y-auto">
                {spreads.map((spread, idx) => (
                  <DropdownMenuItem
                    key={spread.id}
                    onClick={() => setCurrentSpreadIndex(idx)}
                    className={
                      idx === currentSpreadIndex
                        ? "bg-muted font-semibold text-primary"
                        : "text-xs"
                    }
                  >
                    <div className="flex flex-col gap-0.5">
                      <span className="text-[11px] font-medium">{spread.chapter}</span>
                      <span className="text-[10px] text-muted-foreground truncate">
                        {spread.title}
                      </span>
                    </div>
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>

            <span className="text-[11px] text-muted-foreground hidden md:inline">
              คีย์ลัด: <kbd className="px-1 py-0.5 bg-muted rounded border border-border text-[10px]">←</kbd> <kbd className="px-1 py-0.5 bg-muted rounded border border-border text-[10px]">→</kbd> เพื่อพลิกหน้า
            </span>
          </div>
        </div>

        {/* Central Book Area (Two-Page Spread) */}
        <div className="flex-1 min-h-0 bg-muted/20 p-3 sm:p-5 flex items-center justify-center overflow-hidden">
          <div className="w-full h-full max-w-4xl bg-card border border-border rounded-lg shadow-xl flex relative overflow-hidden">
            {/* Center Spine Shadow Simulation */}
            <div className="pointer-events-none absolute inset-y-0 left-1/2 -translate-x-1/2 w-8 z-10 flex justify-center">
              <div className="w-full h-full bg-gradient-to-r from-black/10 via-black/25 to-black/10 dark:from-black/30 dark:via-black/60 dark:to-black/30 opacity-70" />
              <div className="w-[1px] h-full bg-border/60 absolute" />
            </div>

            {/* Left Page */}
            <div className="w-1/2 h-full flex flex-col p-5 sm:p-6 overflow-y-auto border-r border-border/50 relative bg-card">
              <div className="pb-3 border-b border-border/60 mb-3">
                <span className="text-[10px] font-semibold uppercase tracking-wider text-primary block">
                  {currentSpread.chapter}
                </span>
                <h3 className="text-sm font-bold text-foreground mt-0.5">
                  {currentSpread.leftPage.title}
                </h3>
                {currentSpread.leftPage.subtitle && (
                  <p className="text-[11px] text-muted-foreground mt-0.5">
                    {currentSpread.leftPage.subtitle}
                  </p>
                )}
              </div>

              {/* Page Content */}
              <div className="flex-1 min-h-0">
                {currentSpread.leftPage.content}
              </div>

              {/* Page Number Footer */}
              <div className="pt-3 border-t border-border/40 mt-auto text-center">
                <span className="text-[10px] font-mono text-muted-foreground">
                  — หน้า {currentSpread.leftPage.pageNumber} —
                </span>
              </div>
            </div>

            {/* Right Page */}
            <div className="w-1/2 h-full flex flex-col p-5 sm:p-6 overflow-y-auto relative bg-card">
              <div className="pb-3 border-b border-border/60 mb-3">
                <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground block">
                  {currentSpread.title}
                </span>
                <h3 className="text-sm font-bold text-foreground mt-0.5">
                  {currentSpread.rightPage.title}
                </h3>
                {currentSpread.rightPage.subtitle && (
                  <p className="text-[11px] text-muted-foreground mt-0.5">
                    {currentSpread.rightPage.subtitle}
                  </p>
                )}
              </div>

              {/* Page Content */}
              <div className="flex-1 min-h-0">
                {currentSpread.rightPage.content}
              </div>

              {/* Page Number Footer */}
              <div className="pt-3 border-t border-border/40 mt-auto text-center">
                <span className="text-[10px] font-mono text-muted-foreground">
                  — หน้า {currentSpread.rightPage.pageNumber} —
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* Bottom Scroller & Navigation Bar */}
        <div className="h-16 border-t border-border bg-popover/80 px-4 sm:px-6 flex items-center justify-between gap-4 shrink-0 select-none">
          {/* Previous Page Button */}
          <Button
            variant="outline"
            size="sm"
            onClick={handlePrev}
            disabled={currentSpreadIndex === 0}
            className="h-8 gap-1 text-xs shrink-0"
          >
            <ChevronLeft className="w-4 h-4" />
            <span className="hidden sm:inline">หน้าก่อนหน้า</span>
          </Button>

          {/* Center Scroller (Slider & Indicator) */}
          <div className="flex-1 max-w-md flex flex-col items-center gap-1">
            <div className="w-full flex items-center gap-3">
              {/* @ui-allow-native */}
              <input
                type="range"
                min={0}
                max={totalSpreads - 1}
                value={currentSpreadIndex}
                onChange={(e) => setCurrentSpreadIndex(Number(e.target.value))}
                className="w-full accent-primary h-1.5 bg-muted rounded-lg cursor-pointer"
                title={`เลื่อนไปคู่หน้าที่ ${currentSpreadIndex + 1}`}
              />
            </div>
            <div className="text-[11px] text-muted-foreground flex items-center gap-1.5">
              <span className="font-medium text-foreground">
                หน้า {currentSpread.leftPage.pageNumber}-{currentSpread.rightPage.pageNumber}
              </span>
              <span>จากทั้งหมด {totalPages} หน้า</span>
              <span className="hidden md:inline">({currentSpread.chapter})</span>
            </div>
          </div>

          {/* Next Page Button */}
          <Button
            variant="outline"
            size="sm"
            onClick={handleNext}
            disabled={currentSpreadIndex === totalSpreads - 1}
            className="h-8 gap-1 text-xs shrink-0"
          >
            <span className="hidden sm:inline">หน้าถัดไป</span>
            <ChevronRight className="w-4 h-4" />
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
};
