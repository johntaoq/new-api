from pathlib import Path
from PIL import Image, ImageDraw, ImageFont
import math


OUT_DIR = Path(r"C:\LocalData\lab\new-api\docs\diagrams")
OUT_DIR.mkdir(parents=True, exist_ok=True)
OUT_PATH = OUT_DIR / "admin-negative-adjustment-dataflow.png"

WIDTH, HEIGHT = 2200, 2900

BG = "#07111f"
PANEL = "#0d1b2f"
PANEL2 = "#12253f"
TEXT = "#e9f2ff"
MUTED = "#9ab4d6"
LINE = "#6dd3ff"
ACCENT_GREEN = "#57e3a0"
ACCENT_BLUE = "#8aa4ff"
ACCENT_YELLOW = "#ffd166"
ACCENT_RED = "#ff7f96"
ACCENT_CYAN = "#8df0ff"
WHITE = "#ffffff"

FONT_PATH = r"C:\Windows\Fonts\msyh.ttc"
FONT_TITLE = ImageFont.truetype(FONT_PATH, 54)
FONT_SUB = ImageFont.truetype(FONT_PATH, 28)
FONT_BOX = ImageFont.truetype(FONT_PATH, 30)
FONT_SMALL = ImageFont.truetype(FONT_PATH, 24)
FONT_MINI = ImageFont.truetype(FONT_PATH, 22)


def text_block(draw, xy, content, font, fill, max_chars, line_gap=8):
    x, y = xy
    lines = []
    for para in content.split("\n"):
        para = para.strip()
        if not para:
            lines.append("")
            continue
        while len(para) > max_chars:
            lines.append(para[:max_chars])
            para = para[max_chars:]
        lines.append(para)
    current_y = y
    for line_text in lines:
        draw.text((x, current_y), line_text, font=font, fill=fill)
        current_y += font.size + line_gap
    return current_y - y


def draw_box(draw, x1, y1, x2, y2, title, body, outline, fill_box):
    draw.rounded_rectangle((x1, y1, x2, y2), radius=28, outline=outline, width=4, fill=fill_box)
    draw.text((x1 + 24, y1 + 20), title, font=FONT_BOX, fill=WHITE)
    text_block(draw, (x1 + 24, y1 + 72), body, FONT_SMALL, TEXT, 24, 6)


def draw_arrow(draw, p1, p2, color=LINE, width=6, head=16):
    x1, y1 = p1
    x2, y2 = p2
    draw.line((x1, y1, x2, y2), fill=color, width=width)
    angle = math.atan2(y2 - y1, x2 - x1)
    a1 = angle + math.pi * 0.9
    a2 = angle - math.pi * 0.9
    p3 = (x2 + head * math.cos(a1), y2 + head * math.sin(a1))
    p4 = (x2 + head * math.cos(a2), y2 + head * math.sin(a2))
    draw.polygon([p2, p3, p4], fill=color)


def draw_label(draw, x, y, content, color):
    draw.rounded_rectangle((x, y, x + 300, y + 46), radius=18, fill=color)
    draw.text((x + 18, y + 8), content, font=FONT_MINI, fill=BG)


image = Image.new("RGB", (WIDTH, HEIGHT), BG)
draw = ImageDraw.Draw(image)

for i, color in enumerate(["#0d2038", "#0a1a2d", "#081525"]):
    padding = 40 + i * 40
    draw.rounded_rectangle((padding, padding, WIDTH - padding, HEIGHT - padding), radius=42, outline=color, width=3)

margin_x = 90
draw.text((margin_x, 60), "管理员负数调账数据流", font=FONT_TITLE, fill=TEXT)
draw.text((margin_x, 128), "Admin Negative Quota Adjustment Data Flow", font=FONT_SUB, fill=MUTED)

draw_box(
    draw, 110, 240, 720, 380,
    "1. 管理后台入口",
    "管理员提交负数调账\nPOST /api/user/:id/quota_adjust",
    ACCENT_BLUE, PANEL,
)
draw_box(
    draw, 110, 460, 720, 650,
    "2. 控制器校验",
    "controller.AdjustUserQuotaByAdmin\n校验目标用户权限\n校验 paid / gift 桶\n负数默认 source_type = system_adjustment",
    ACCENT_BLUE, PANEL,
)
draw_box(
    draw, 110, 740, 720, 970,
    "3. 审计事务入口",
    "model.AdjustUserQuotaWithAudit\n先锁用户并抓 before 快照\n快照字段：quota / paid_quota / gift_quota",
    ACCENT_CYAN, PANEL,
)
draw_box(
    draw, 110, 1060, 720, 1310,
    "4. 真正扣库存",
    "adjustUserQuotaWithLedgerTx\n调用 consumeUserQuotaByFundingTypeTx\n只按指定 bucket 扣，不混扣",
    ACCENT_YELLOW, PANEL,
)
draw_box(
    draw, 110, 1400, 720, 1700,
    "5. users 表余额回写",
    "更新 users.paid_quota\n更新 users.gift_quota\n更新 users.quota\n如果余额不足则整笔失败回滚",
    ACCENT_RED, PANEL,
)
draw_box(
    draw, 110, 1790, 720, 2080,
    "6. UserBalanceLedger",
    "新增一条负向余额流水\nentry_type = adjustment\ndirection = debit\namount_quota = 负数\namount_usd = 负数",
    ACCENT_GREEN, PANEL,
)
draw_box(
    draw, 110, 2170, 720, 2460,
    "7. FinancialAuditLog",
    "新增结构化财务审计\nbefore_json = 调前余额\nafter_json = 调后余额 + delta + bucket + source_type\nremark / operator 一起落表",
    ACCENT_CYAN, PANEL,
)
draw_box(
    draw, 110, 2550, 720, 2800,
    "8. 后台管理日志",
    "额外写 logs\n类型 = LogTypeManage\n用于后台行为留痕\n但不是正式账单来源",
    ACCENT_BLUE, PANEL,
)

draw_box(
    draw, 860, 240, 2060, 660,
    "A. user_quota_fundings（库存层）",
    "负数调账不会新增 funding。\n它只会按创建时间顺序，扣减已有 funding 的 remaining_quota。\n所以这里体现为：库存余额减少，而不是新增一条负 funding。\n\n影响字段：\n- funding_type\n- source_type\n- remaining_quota\n- updated_at",
    ACCENT_YELLOW, PANEL2,
)
draw_box(
    draw, 860, 760, 2060, 1220,
    "B. user_balance_ledgers（正式非消费流水）",
    "这是负数调账的正式账务流水来源。\n\n关键字段：\n- bucket_type = paid | gift\n- entry_type = adjustment\n- direction = debit\n- amount_quota = 负数\n- amount_usd = 负数\n- source_type\n- external_ref\n- operator_user_id\n- operator_username_snapshot\n- remark",
    ACCENT_GREEN, PANEL2,
)
draw_box(
    draw, 860, 1320, 2060, 1820,
    "C. customer_monthly_statement_items（月账单明细）",
    "后续生成月账单时，会从 user_balance_ledgers 读取这条记录。\n\n账单侧表现：\n- entry_type = adjustment\n- source_table = user_balance_ledgers\n- display / USD 金额都是负数\n- request_id 取 external_ref，若无则退回 source_name\n- content_summary = 来源标签 + remark",
    ACCENT_CYAN, PANEL2,
)
draw_box(
    draw, 860, 1920, 2060, 2420,
    "D. 不会写入 channel_cost_ledgers",
    "因为这不是模型调用消费，也不是消费退款。\n\n所以：\n- 不计入消费流水层\n- 不计入渠道成本\n- 不会伪装成 consume / refund\n\n它只属于非消费类余额调整。",
    ACCENT_RED, PANEL2,
)
draw_box(
    draw, 860, 2520, 2060, 2800,
    "E. 失败语义",
    "如果扣减值大于该桶当前余额：\n- 整笔事务失败\n- users 不更新\n- balance ledger 不写\n- audit log 不写\n\n也就是说，不会成功把用户余额扣成负数。",
    ACCENT_BLUE, PANEL2,
)

draw_arrow(draw, (415, 380), (415, 460))
draw_arrow(draw, (415, 650), (415, 740))
draw_arrow(draw, (415, 970), (415, 1060))
draw_arrow(draw, (415, 1310), (415, 1400))
draw_arrow(draw, (415, 1700), (415, 1790))
draw_arrow(draw, (415, 2080), (415, 2170))
draw_arrow(draw, (415, 2460), (415, 2550))

draw_arrow(draw, (720, 1185), (860, 450), ACCENT_YELLOW)
draw_arrow(draw, (720, 1935), (860, 980), ACCENT_GREEN)
draw_arrow(draw, (720, 2315), (860, 1570), ACCENT_CYAN)
draw_arrow(draw, (1460, 1220), (1460, 1320), ACCENT_CYAN)

draw_label(draw, 520, 1085, "按桶扣库存", ACCENT_YELLOW)
draw_label(draw, 520, 1815, "正式流水", ACCENT_GREEN)
draw_label(draw, 520, 2195, "结构化审计", ACCENT_CYAN)
draw_label(draw, 1510, 1238, "生成月账单时读取", "#b8c6ff")

draw.text(
    (90, 2840),
    "口径总结：负数调账 = 扣库存 + 记 adjustment 流水 + 记审计，不进入渠道消费成本。",
    font=FONT_SUB,
    fill=MUTED,
)

image.save(OUT_PATH)
print(OUT_PATH)
