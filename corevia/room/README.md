# room 契约草图

对应领域事实见 corevia `backend/app/room/CONTEXT.md` 与 ADR 0072–0074。本目录是 **Room 契约草图**，尚未 `make gen-go` 发版。

## 分层

| 包 | 调用方 | 职责 |
|----|--------|------|
| `room/v1` | room 服务 / 其它后端 | 房间、场次、麦位、处置、Closure participant |
| `room/internal/v1` | Gift / 受信任 BFF | 房内送礼门闸、场次系统提示（进三方直播群） |
| `gamoji/room/v1` | Gamoji C 端（经 gamoji-bff） | `/api/v1/**` REST；App 由 BFF 固定注入 |
| `admin/room/v1` | Admin（经 admin BFF） | `/admin/v1/**` 查询与强处置 |

## 与既有契约的关系

- **送礼**：仍走 `gift.v1` `SendGift` / `SendBackpackGift`，`scene_type=ROOM`，`scene_id=<room_session_id>`；Recipent 由 `RoomGiftGateService.ValidateSessionGiftTarget` 在发送前校验（任意在场成员、禁自送、被处置不可收）。
- **消息**：Room 不暴露消息 RPC；`LiveGroupBinding` 只给 provider + group_id。
- **RTC**：Join/Open 返回 `rtc_token` + `rtc_channel_id`；踢人/静音/关房由 Room 域服务驱动，不暴露厂商 SDK 类型。
- **User Deletion**：`RoomClosureService` 与 gift/wallet/payment/ops 的 participant 契约同构。

## 一期范围备忘

- 仅 `ROOM_TYPE_CHAT` 能力；`GAMING`/`KTV`/`SHOW` 枚举预留。
- 无举报、无消息读写、无多端在场、无门票。
