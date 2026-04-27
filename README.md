# Albert Morning Agent

บอทส่งสรุปข่าวเช้าอัตโนมัติผ่าน LINE — ดึงข่าวจาก Google News, สรุปด้วย AI (Groq / LLaMA 3.3 70B), แล้วส่งเข้า LINE ทุกเช้า 7:00 น. (เวลาไทย)

---

## ภาพรวมระบบ

```
Google News RSS
      │
      ▼
  news.FetchRSS          ← ดึงพาดหัวข่าวล่าสุด (สูงสุด 10 ข่าวต่อหัวข้อ)
      │
      ▼
  gemini.Summarize        ← ส่ง prompt ไปที่ Groq API (LLaMA 3.3 70B)
      │                      → ได้สรุปเป็นภาษาไทย ไม่เกิน 5 ประเด็น
      ▼
  line.Send               ← ส่งข้อความเข้า LINE
      │
      ├── Push mode      → ส่งหาผู้ใช้คนเดียว (LINE_USER_ID)
      └── Broadcast mode → ส่งหาทุกคนที่ Add บอทเป็นเพื่อน
```

---

## ฟีเจอร์

### หัวข้อข่าวรายวัน (ทุกวัน)

| หัวข้อ | แหล่งข้อมูล |
|--------|------------|
| ข่าว AI และเทคโนโลยี | Google News (ค้นหา: artificial intelligence AI) |
| หุ้นไทย (SET) | Google News (ค้นหา: SET index thailand stock) |
| หุ้นอเมริกา (Nasdaq / S&P500) | Google News (ค้นหา: stock market nasdaq S&P500) |

### หัวข้อพิเศษ (เฉพาะวันที่ 1 และ 16 ของเดือน)

| หัวข้อ | แหล่งข้อมูล |
|--------|------------|
| ผลหวยไทย (สลากกินแบ่งรัฐบาล) | Google News (ค้นหา: ผลสลากกินแบ่งรัฐบาล หวย) |

### รูปแบบข้อความที่ส่ง

```
🤖 *ข่าว AI* ประจำวัน 27/04/2026

📌 ...
📌 ...
📌 ...
```

---

## โครงสร้างโปรเจกต์

```
albert-morning-agent/
├── cmd/
│   └── agent/
│       └── main.go          # Entry point — orchestrate ทั้งกระบวนการ
├── internal/
│   ├── config/
│   │   └── config.go        # โหลด env vars และ validate
│   ├── news/
│   │   └── fetcher.go       # ดึง RSS feed และแปลงเป็น list พาดหัว
│   ├── gemini/
│   │   └── client.go        # เรียก Groq API (LLaMA 3.3 70B) เพื่อสรุปข่าว
│   └── line/
│       └── client.go        # ส่งข้อความผ่าน LINE Messaging API
├── .github/
│   └── workflows/
│       └── morning.yml      # GitHub Actions — รันทุกวัน 7:00 น. (Bangkok)
├── .env                     # env vars สำหรับ local dev (ห้าม commit)
├── go.mod
└── go.sum
```

---

## การตั้งค่า Environment Variables

| ตัวแปร | จำเป็น | คำอธิบาย |
|--------|--------|---------|
| `GROQ_API_KEY` | ใช่ | API Key จาก Groq Console |
| `LINE_CHANNEL_ACCESS_TOKEN` | ใช่ | Channel Access Token จาก LINE Developers Console |
| `LINE_USER_ID` | เฉพาะ Push mode | User ID ของผู้รับข้อความ (ขึ้นต้นด้วย `U`) |
| `LINE_BROADCAST` | ไม่ | ตั้งเป็น `true` เพื่อส่งหาทุกคนที่ Add บอท (ค่าเริ่มต้น: `false`) |

### Push mode vs Broadcast mode

**Push mode** (ค่าเริ่มต้น) — ส่งหาผู้ใช้คนเดียวที่ระบุใน `LINE_USER_ID`

```env
LINE_BROADCAST=false
LINE_USER_ID=Uxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

**Broadcast mode** — ส่งหาทุกคนที่ Add บอทเป็นเพื่อน ไม่ต้องระบุ `LINE_USER_ID`

```env
LINE_BROADCAST=true
```

> **หมายเหตุ:** Broadcast API ใช้ได้เฉพาะ LINE Messaging API channel และอาจมีค่าใช้จ่ายตาม plan ที่ใช้อยู่

---

## การติดตั้งและรันแบบ Local

### ความต้องการของระบบ

- Go 1.22 ขึ้นไป
- Groq API Key
- LINE Messaging API Channel Access Token

### ขั้นตอน

**1. Clone โปรเจกต์**

```bash
git clone https://github.com/claymorepriscilla/albert-morning-agent.git
cd albert-morning-agent
```

**2. สร้างไฟล์ `.env`**

```env
GROQ_API_KEY=YOUR_GROQ_API_KEY_HERE
LINE_CHANNEL_ACCESS_TOKEN=YOUR_LINE_CHANNEL_ACCESS_TOKEN_HERE
LINE_USER_ID=YOUR_LINE_USER_ID_HERE
LINE_BROADCAST=false
```

**3. รัน**

```bash
go run ./cmd/agent
```

---

## การ Deploy ด้วย GitHub Actions

บอทรันอัตโนมัติผ่าน GitHub Actions ทุกวัน 7:00 น. เวลาไทย (00:00 UTC)

### ตั้งค่า GitHub Secrets

ไปที่ **Settings → Secrets and variables → Actions** แล้วเพิ่ม:

| Secret | ค่า |
|--------|-----|
| `GROQ_API_KEY` | API Key จาก Groq |
| `LINE_CHANNEL_ACCESS_TOKEN` | Channel Access Token จาก LINE |
| `LINE_USER_ID` | User ID ผู้รับ (เฉพาะ Push mode) |

> `LINE_BROADCAST` ไม่ต้องเก็บใน Secrets เพราะไม่ใช่ข้อมูลลับ — ตั้งตรงใน workflow file ได้เลย

### รันทดสอบด้วยตนเอง

ไปที่ **Actions → Morning Agent → Run workflow** แล้วกด **Run workflow**

---

## Flow การทำงานแบบละเอียด

```
1. โหลด config จาก env vars และ validate ความครบถ้วน

2. สำหรับแต่ละหัวข้อ (AI / หุ้นไทย / หุ้นอเมริกา):
   a. FetchRSS  — ดึงพาดหัวข่าวล่าสุดสูงสุด 10 ข้อจาก Google News RSS
   b. Summarize — ส่ง prompt ไปที่ Groq API (LLaMA 3.3 70B)
                  prompt: "สรุปข่าว{topic} กระชับ ไม่เกิน 5 ประเด็น ใช้รูปแบบ 📌"
   c. Send      — ส่งข้อความสรุปเข้า LINE (Push หรือ Broadcast ตาม config)

   หากเกิด error ในขั้นตอนใด จะ log และข้ามไปหัวข้อถัดไป (best-effort)

3. ถ้าวันที่ปัจจุบันคือวันที่ 1 หรือ 16 ของเดือน:
   → ทำซ้ำขั้นตอนเดิมสำหรับหัวข้อผลหวยไทย

4. จบการทำงาน — log "Morning Agent completed."
```

---

## Dependencies

| Package | การใช้งาน |
|---------|----------|
| `github.com/joho/godotenv` | โหลด `.env` file สำหรับ local dev |
| `github.com/mmcdole/gofeed` | Parse RSS feed จาก Google News |

AI Engine: **Groq API** — โมเดล `llama-3.3-70b-versatile`

---

## Security

- ไฟล์ `.env` ต้องอยู่ใน `.gitignore` เสมอ — ห้าม commit ขึ้น repository
- ใช้ GitHub Secrets สำหรับ credentials ทั้งหมดบน CI/CD
- LINE Channel Access Token มีสิทธิ์ส่งข้อความในนามบอท — เก็บรักษาให้ดี
