-- Seed 500 non-user business records for go-order-management-system.
--
-- 可靠性说明：
-- 1. 本脚本不插入 users / user_roles / roles，也不依赖现有用户。
-- 2. 只写入商品、库存、库存流水三类业务表，避免订单外键导致导入失败。
-- 3. 使用 [seed-500] 标记清理并重建本批数据，可在本地/Navicat 中重复执行。
--
-- 目标数据量：
--   products:             125
--   product_inventories:  125
--   stock_logs:           250  (125 初始化库存 + 125 手动入库)
--   total:                500
--
-- Navicat 使用方式：
--   先在 Navicat 左侧选中当前项目数据库，例如 go_order_management_system，
--   再打开并运行整个脚本。执行结束后看最后一条 SELECT 的 total_rows 是否为 500。

SET NAMES utf8mb4;
SET @old_sql_safe_updates := @@SQL_SAFE_UPDATES;
SET SQL_SAFE_UPDATES = 0;

DROP TEMPORARY TABLE IF EXISTS _seed_products_500;
CREATE TEMPORARY TABLE _seed_products_500 (
    seed_no INT NOT NULL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500) NOT NULL,
    price_fen BIGINT NOT NULL,
    status TINYINT NOT NULL,
    base_stock BIGINT NOT NULL,
    manual_add BIGINT NOT NULL,
    final_stock BIGINT NOT NULL
) ENGINE = MEMORY DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_general_ci;

INSERT INTO _seed_products_500 (seed_no, name, description, price_fen, status, base_stock, manual_add, final_stock) VALUES
    (1, 'Seed Product 001 - 机械键盘', '[seed-500] Navicat 可重复导入的非用户商品库存数据 001', 1396, 1, 85, 12, 97),
    (2, 'Seed Product 002 - 无线鼠标', '[seed-500] Navicat 可重复导入的非用户商品库存数据 002', 1493, 1, 90, 14, 104),
    (3, 'Seed Product 003 - 蓝牙耳机', '[seed-500] Navicat 可重复导入的非用户商品库存数据 003', 1590, 1, 95, 16, 111),
    (4, 'Seed Product 004 - USB-C 扩展坞', '[seed-500] Navicat 可重复导入的非用户商品库存数据 004', 1687, 1, 100, 18, 118),
    (5, 'Seed Product 005 - 显示器支架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 005', 1784, 1, 105, 20, 125),
    (6, 'Seed Product 006 - 笔记本支架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 006', 1881, 1, 110, 22, 132),
    (7, 'Seed Product 007 - 移动电源', '[seed-500] Navicat 可重复导入的非用户商品库存数据 007', 1978, 1, 115, 24, 139),
    (8, 'Seed Product 008 - 便携显示器', '[seed-500] Navicat 可重复导入的非用户商品库存数据 008', 2075, 1, 120, 26, 146),
    (9, 'Seed Product 009 - 智能台灯', '[seed-500] Navicat 可重复导入的非用户商品库存数据 009', 2172, 1, 125, 10, 135),
    (10, 'Seed Product 010 - 桌面收纳架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 010', 2269, 1, 130, 12, 142),
    (11, 'Seed Product 011 - 电脑背包', '[seed-500] Navicat 可重复导入的非用户商品库存数据 011', 2366, 1, 135, 14, 149),
    (12, 'Seed Product 012 - 网线套装', '[seed-500] Navicat 可重复导入的非用户商品库存数据 012', 2463, 1, 140, 16, 156),
    (13, 'Seed Product 013 - 固态硬盘', '[seed-500] Navicat 可重复导入的非用户商品库存数据 013', 2560, 2, 145, 18, 163),
    (14, 'Seed Product 014 - 内存条', '[seed-500] Navicat 可重复导入的非用户商品库存数据 014', 2657, 1, 150, 20, 170),
    (15, 'Seed Product 015 - 散热风扇', '[seed-500] Navicat 可重复导入的非用户商品库存数据 015', 2754, 1, 155, 22, 177),
    (16, 'Seed Product 016 - 机械键盘', '[seed-500] Navicat 可重复导入的非用户商品库存数据 016', 2851, 1, 160, 24, 184),
    (17, 'Seed Product 017 - 无线鼠标', '[seed-500] Navicat 可重复导入的非用户商品库存数据 017', 2948, 1, 80, 26, 106),
    (18, 'Seed Product 018 - 蓝牙耳机', '[seed-500] Navicat 可重复导入的非用户商品库存数据 018', 3045, 1, 85, 10, 95),
    (19, 'Seed Product 019 - USB-C 扩展坞', '[seed-500] Navicat 可重复导入的非用户商品库存数据 019', 3142, 1, 90, 12, 102),
    (20, 'Seed Product 020 - 显示器支架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 020', 3239, 1, 98, 14, 112),
    (21, 'Seed Product 021 - 笔记本支架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 021', 3336, 1, 103, 16, 119),
    (22, 'Seed Product 022 - 移动电源', '[seed-500] Navicat 可重复导入的非用户商品库存数据 022', 3433, 1, 108, 18, 126),
    (23, 'Seed Product 023 - 便携显示器', '[seed-500] Navicat 可重复导入的非用户商品库存数据 023', 3530, 1, 113, 20, 133),
    (24, 'Seed Product 024 - 智能台灯', '[seed-500] Navicat 可重复导入的非用户商品库存数据 024', 3627, 1, 118, 22, 140),
    (25, 'Seed Product 025 - 桌面收纳架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 025', 3724, 1, 123, 24, 147),
    (26, 'Seed Product 026 - 电脑背包', '[seed-500] Navicat 可重复导入的非用户商品库存数据 026', 3821, 2, 128, 26, 154),
    (27, 'Seed Product 027 - 网线套装', '[seed-500] Navicat 可重复导入的非用户商品库存数据 027', 3918, 1, 133, 10, 143),
    (28, 'Seed Product 028 - 固态硬盘', '[seed-500] Navicat 可重复导入的非用户商品库存数据 028', 4015, 1, 138, 12, 150),
    (29, 'Seed Product 029 - 内存条', '[seed-500] Navicat 可重复导入的非用户商品库存数据 029', 4112, 1, 143, 14, 157),
    (30, 'Seed Product 030 - 散热风扇', '[seed-500] Navicat 可重复导入的非用户商品库存数据 030', 4209, 1, 148, 16, 164),
    (31, 'Seed Product 031 - 机械键盘', '[seed-500] Navicat 可重复导入的非用户商品库存数据 031', 4306, 1, 153, 18, 171),
    (32, 'Seed Product 032 - 无线鼠标', '[seed-500] Navicat 可重复导入的非用户商品库存数据 032', 4403, 1, 158, 20, 178),
    (33, 'Seed Product 033 - 蓝牙耳机', '[seed-500] Navicat 可重复导入的非用户商品库存数据 033', 4500, 1, 163, 22, 185),
    (34, 'Seed Product 034 - USB-C 扩展坞', '[seed-500] Navicat 可重复导入的非用户商品库存数据 034', 4597, 1, 83, 24, 107),
    (35, 'Seed Product 035 - 显示器支架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 035', 4694, 1, 88, 26, 114),
    (36, 'Seed Product 036 - 笔记本支架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 036', 4791, 1, 93, 10, 103),
    (37, 'Seed Product 037 - 移动电源', '[seed-500] Navicat 可重复导入的非用户商品库存数据 037', 4888, 1, 98, 12, 110),
    (38, 'Seed Product 038 - 便携显示器', '[seed-500] Navicat 可重复导入的非用户商品库存数据 038', 4985, 1, 103, 14, 117),
    (39, 'Seed Product 039 - 智能台灯', '[seed-500] Navicat 可重复导入的非用户商品库存数据 039', 5082, 2, 108, 16, 124),
    (40, 'Seed Product 040 - 桌面收纳架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 040', 5179, 1, 116, 18, 134),
    (41, 'Seed Product 041 - 电脑背包', '[seed-500] Navicat 可重复导入的非用户商品库存数据 041', 5276, 1, 121, 20, 141),
    (42, 'Seed Product 042 - 网线套装', '[seed-500] Navicat 可重复导入的非用户商品库存数据 042', 5373, 1, 126, 22, 148),
    (43, 'Seed Product 043 - 固态硬盘', '[seed-500] Navicat 可重复导入的非用户商品库存数据 043', 5470, 1, 131, 24, 155),
    (44, 'Seed Product 044 - 内存条', '[seed-500] Navicat 可重复导入的非用户商品库存数据 044', 5567, 1, 136, 26, 162),
    (45, 'Seed Product 045 - 散热风扇', '[seed-500] Navicat 可重复导入的非用户商品库存数据 045', 5664, 1, 141, 10, 151),
    (46, 'Seed Product 046 - 机械键盘', '[seed-500] Navicat 可重复导入的非用户商品库存数据 046', 5761, 1, 146, 12, 158),
    (47, 'Seed Product 047 - 无线鼠标', '[seed-500] Navicat 可重复导入的非用户商品库存数据 047', 5858, 1, 151, 14, 165),
    (48, 'Seed Product 048 - 蓝牙耳机', '[seed-500] Navicat 可重复导入的非用户商品库存数据 048', 5955, 1, 156, 16, 172),
    (49, 'Seed Product 049 - USB-C 扩展坞', '[seed-500] Navicat 可重复导入的非用户商品库存数据 049', 6052, 1, 161, 18, 179),
    (50, 'Seed Product 050 - 显示器支架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 050', 6149, 1, 166, 20, 186),
    (51, 'Seed Product 051 - 笔记本支架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 051', 6246, 1, 86, 22, 108),
    (52, 'Seed Product 052 - 移动电源', '[seed-500] Navicat 可重复导入的非用户商品库存数据 052', 6343, 2, 91, 24, 115),
    (53, 'Seed Product 053 - 便携显示器', '[seed-500] Navicat 可重复导入的非用户商品库存数据 053', 6440, 1, 96, 26, 122),
    (54, 'Seed Product 054 - 智能台灯', '[seed-500] Navicat 可重复导入的非用户商品库存数据 054', 6537, 1, 101, 10, 111),
    (55, 'Seed Product 055 - 桌面收纳架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 055', 6634, 1, 106, 12, 118),
    (56, 'Seed Product 056 - 电脑背包', '[seed-500] Navicat 可重复导入的非用户商品库存数据 056', 6731, 1, 111, 14, 125),
    (57, 'Seed Product 057 - 网线套装', '[seed-500] Navicat 可重复导入的非用户商品库存数据 057', 6828, 1, 116, 16, 132),
    (58, 'Seed Product 058 - 固态硬盘', '[seed-500] Navicat 可重复导入的非用户商品库存数据 058', 6925, 1, 121, 18, 139),
    (59, 'Seed Product 059 - 内存条', '[seed-500] Navicat 可重复导入的非用户商品库存数据 059', 7022, 1, 126, 20, 146),
    (60, 'Seed Product 060 - 散热风扇', '[seed-500] Navicat 可重复导入的非用户商品库存数据 060', 7119, 1, 134, 22, 156),
    (61, 'Seed Product 061 - 机械键盘', '[seed-500] Navicat 可重复导入的非用户商品库存数据 061', 7216, 1, 139, 24, 163),
    (62, 'Seed Product 062 - 无线鼠标', '[seed-500] Navicat 可重复导入的非用户商品库存数据 062', 7313, 1, 144, 26, 170),
    (63, 'Seed Product 063 - 蓝牙耳机', '[seed-500] Navicat 可重复导入的非用户商品库存数据 063', 7410, 1, 149, 10, 159),
    (64, 'Seed Product 064 - USB-C 扩展坞', '[seed-500] Navicat 可重复导入的非用户商品库存数据 064', 7507, 1, 154, 12, 166),
    (65, 'Seed Product 065 - 显示器支架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 065', 7604, 2, 159, 14, 173),
    (66, 'Seed Product 066 - 笔记本支架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 066', 7701, 1, 164, 16, 180),
    (67, 'Seed Product 067 - 移动电源', '[seed-500] Navicat 可重复导入的非用户商品库存数据 067', 7798, 1, 169, 18, 187),
    (68, 'Seed Product 068 - 便携显示器', '[seed-500] Navicat 可重复导入的非用户商品库存数据 068', 7895, 1, 89, 20, 109),
    (69, 'Seed Product 069 - 智能台灯', '[seed-500] Navicat 可重复导入的非用户商品库存数据 069', 7992, 1, 94, 22, 116),
    (70, 'Seed Product 070 - 桌面收纳架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 070', 8089, 1, 99, 24, 123),
    (71, 'Seed Product 071 - 电脑背包', '[seed-500] Navicat 可重复导入的非用户商品库存数据 071', 8186, 1, 104, 26, 130),
    (72, 'Seed Product 072 - 网线套装', '[seed-500] Navicat 可重复导入的非用户商品库存数据 072', 8283, 1, 109, 10, 119),
    (73, 'Seed Product 073 - 固态硬盘', '[seed-500] Navicat 可重复导入的非用户商品库存数据 073', 8380, 1, 114, 12, 126),
    (74, 'Seed Product 074 - 内存条', '[seed-500] Navicat 可重复导入的非用户商品库存数据 074', 8477, 1, 119, 14, 133),
    (75, 'Seed Product 075 - 散热风扇', '[seed-500] Navicat 可重复导入的非用户商品库存数据 075', 8574, 1, 124, 16, 140),
    (76, 'Seed Product 076 - 机械键盘', '[seed-500] Navicat 可重复导入的非用户商品库存数据 076', 8671, 1, 129, 18, 147),
    (77, 'Seed Product 077 - 无线鼠标', '[seed-500] Navicat 可重复导入的非用户商品库存数据 077', 8768, 1, 134, 20, 154),
    (78, 'Seed Product 078 - 蓝牙耳机', '[seed-500] Navicat 可重复导入的非用户商品库存数据 078', 8865, 2, 139, 22, 161),
    (79, 'Seed Product 079 - USB-C 扩展坞', '[seed-500] Navicat 可重复导入的非用户商品库存数据 079', 8962, 1, 144, 24, 168),
    (80, 'Seed Product 080 - 显示器支架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 080', 9059, 1, 152, 26, 178),
    (81, 'Seed Product 081 - 笔记本支架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 081', 9156, 1, 157, 10, 167),
    (82, 'Seed Product 082 - 移动电源', '[seed-500] Navicat 可重复导入的非用户商品库存数据 082', 9253, 1, 162, 12, 174),
    (83, 'Seed Product 083 - 便携显示器', '[seed-500] Navicat 可重复导入的非用户商品库存数据 083', 9350, 1, 167, 14, 181),
    (84, 'Seed Product 084 - 智能台灯', '[seed-500] Navicat 可重复导入的非用户商品库存数据 084', 9447, 1, 172, 16, 188),
    (85, 'Seed Product 085 - 桌面收纳架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 085', 9544, 1, 92, 18, 110),
    (86, 'Seed Product 086 - 电脑背包', '[seed-500] Navicat 可重复导入的非用户商品库存数据 086', 9641, 1, 97, 20, 117),
    (87, 'Seed Product 087 - 网线套装', '[seed-500] Navicat 可重复导入的非用户商品库存数据 087', 9738, 1, 102, 22, 124),
    (88, 'Seed Product 088 - 固态硬盘', '[seed-500] Navicat 可重复导入的非用户商品库存数据 088', 9835, 1, 107, 24, 131),
    (89, 'Seed Product 089 - 内存条', '[seed-500] Navicat 可重复导入的非用户商品库存数据 089', 9932, 1, 112, 26, 138),
    (90, 'Seed Product 090 - 散热风扇', '[seed-500] Navicat 可重复导入的非用户商品库存数据 090', 10029, 1, 117, 10, 127),
    (91, 'Seed Product 091 - 机械键盘', '[seed-500] Navicat 可重复导入的非用户商品库存数据 091', 10126, 2, 122, 12, 134),
    (92, 'Seed Product 092 - 无线鼠标', '[seed-500] Navicat 可重复导入的非用户商品库存数据 092', 10223, 1, 127, 14, 141),
    (93, 'Seed Product 093 - 蓝牙耳机', '[seed-500] Navicat 可重复导入的非用户商品库存数据 093', 10320, 1, 132, 16, 148),
    (94, 'Seed Product 094 - USB-C 扩展坞', '[seed-500] Navicat 可重复导入的非用户商品库存数据 094', 10417, 1, 137, 18, 155),
    (95, 'Seed Product 095 - 显示器支架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 095', 10514, 1, 142, 20, 162),
    (96, 'Seed Product 096 - 笔记本支架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 096', 10611, 1, 147, 22, 169),
    (97, 'Seed Product 097 - 移动电源', '[seed-500] Navicat 可重复导入的非用户商品库存数据 097', 10708, 1, 152, 24, 176),
    (98, 'Seed Product 098 - 便携显示器', '[seed-500] Navicat 可重复导入的非用户商品库存数据 098', 10805, 1, 157, 26, 183),
    (99, 'Seed Product 099 - 智能台灯', '[seed-500] Navicat 可重复导入的非用户商品库存数据 099', 10902, 1, 162, 10, 172),
    (100, 'Seed Product 100 - 桌面收纳架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 100', 10999, 1, 170, 12, 182),
    (101, 'Seed Product 101 - 电脑背包', '[seed-500] Navicat 可重复导入的非用户商品库存数据 101', 11096, 1, 175, 14, 189),
    (102, 'Seed Product 102 - 网线套装', '[seed-500] Navicat 可重复导入的非用户商品库存数据 102', 11193, 1, 95, 16, 111),
    (103, 'Seed Product 103 - 固态硬盘', '[seed-500] Navicat 可重复导入的非用户商品库存数据 103', 11290, 1, 100, 18, 118),
    (104, 'Seed Product 104 - 内存条', '[seed-500] Navicat 可重复导入的非用户商品库存数据 104', 11387, 2, 105, 20, 125),
    (105, 'Seed Product 105 - 散热风扇', '[seed-500] Navicat 可重复导入的非用户商品库存数据 105', 11484, 1, 110, 22, 132),
    (106, 'Seed Product 106 - 机械键盘', '[seed-500] Navicat 可重复导入的非用户商品库存数据 106', 11581, 1, 115, 24, 139),
    (107, 'Seed Product 107 - 无线鼠标', '[seed-500] Navicat 可重复导入的非用户商品库存数据 107', 11678, 1, 120, 26, 146),
    (108, 'Seed Product 108 - 蓝牙耳机', '[seed-500] Navicat 可重复导入的非用户商品库存数据 108', 11775, 1, 125, 10, 135),
    (109, 'Seed Product 109 - USB-C 扩展坞', '[seed-500] Navicat 可重复导入的非用户商品库存数据 109', 11872, 1, 130, 12, 142),
    (110, 'Seed Product 110 - 显示器支架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 110', 11969, 1, 135, 14, 149),
    (111, 'Seed Product 111 - 笔记本支架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 111', 12066, 1, 140, 16, 156),
    (112, 'Seed Product 112 - 移动电源', '[seed-500] Navicat 可重复导入的非用户商品库存数据 112', 12163, 1, 145, 18, 163),
    (113, 'Seed Product 113 - 便携显示器', '[seed-500] Navicat 可重复导入的非用户商品库存数据 113', 12260, 1, 150, 20, 170),
    (114, 'Seed Product 114 - 智能台灯', '[seed-500] Navicat 可重复导入的非用户商品库存数据 114', 12357, 1, 155, 22, 177),
    (115, 'Seed Product 115 - 桌面收纳架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 115', 12454, 1, 160, 24, 184),
    (116, 'Seed Product 116 - 电脑背包', '[seed-500] Navicat 可重复导入的非用户商品库存数据 116', 12551, 1, 165, 26, 191),
    (117, 'Seed Product 117 - 网线套装', '[seed-500] Navicat 可重复导入的非用户商品库存数据 117', 12648, 2, 170, 10, 180),
    (118, 'Seed Product 118 - 固态硬盘', '[seed-500] Navicat 可重复导入的非用户商品库存数据 118', 12745, 1, 175, 12, 187),
    (119, 'Seed Product 119 - 内存条', '[seed-500] Navicat 可重复导入的非用户商品库存数据 119', 12842, 1, 95, 14, 109),
    (120, 'Seed Product 120 - 散热风扇', '[seed-500] Navicat 可重复导入的非用户商品库存数据 120', 12939, 1, 103, 16, 119),
    (121, 'Seed Product 121 - 机械键盘', '[seed-500] Navicat 可重复导入的非用户商品库存数据 121', 13036, 1, 108, 18, 126),
    (122, 'Seed Product 122 - 无线鼠标', '[seed-500] Navicat 可重复导入的非用户商品库存数据 122', 13133, 1, 113, 20, 133),
    (123, 'Seed Product 123 - 蓝牙耳机', '[seed-500] Navicat 可重复导入的非用户商品库存数据 123', 13230, 1, 118, 22, 140),
    (124, 'Seed Product 124 - USB-C 扩展坞', '[seed-500] Navicat 可重复导入的非用户商品库存数据 124', 13327, 1, 123, 24, 147),
    (125, 'Seed Product 125 - 显示器支架', '[seed-500] Navicat 可重复导入的非用户商品库存数据 125', 13424, 1, 128, 26, 154);

START TRANSACTION;

-- 只清理本脚本生成的 seed 数据，不影响手工创建的业务数据。
DELETE FROM stock_logs
WHERE remark LIKE '[seed-500]%';

DELETE pi
FROM product_inventories AS pi
INNER JOIN products AS p ON p.id = pi.product_id
WHERE p.description LIKE '[seed-500]%';

DELETE FROM products
WHERE description LIKE '[seed-500]%';

INSERT INTO products (name, description, price_fen, status, created_at, updated_at)
SELECT name, description, price_fen, status, NOW(3), NOW(3)
FROM _seed_products_500
ORDER BY seed_no;

DROP TEMPORARY TABLE IF EXISTS _seed_product_ids_500;
CREATE TEMPORARY TABLE _seed_product_ids_500 AS
SELECT sp.seed_no,
       p.id AS product_id,
       sp.name,
       sp.base_stock,
       sp.manual_add,
       sp.final_stock
FROM _seed_products_500 AS sp
INNER JOIN products AS p
        ON p.name = sp.name
       AND p.description = sp.description;

INSERT INTO product_inventories (product_id, stock_quantity, created_at, updated_at)
SELECT product_id, final_stock, NOW(3), NOW(3)
FROM _seed_product_ids_500
ORDER BY seed_no;

INSERT INTO stock_logs (product_id, change_quantity, before_quantity, after_quantity, biz_type, biz_id, remark, created_at)
SELECT product_id,
       base_stock,
       0,
       base_stock,
       1,
       NULL,
       CONCAT('[seed-500] 初始化库存：', name),
       DATE_SUB(NOW(3), INTERVAL seed_no MINUTE)
FROM _seed_product_ids_500
ORDER BY seed_no;

INSERT INTO stock_logs (product_id, change_quantity, before_quantity, after_quantity, biz_type, biz_id, remark, created_at)
SELECT product_id,
       manual_add,
       base_stock,
       final_stock,
       2,
       NULL,
       CONCAT('[seed-500] 手动入库：', name),
       NOW(3)
FROM _seed_product_ids_500
ORDER BY seed_no;

COMMIT;

SET SQL_SAFE_UPDATES = @old_sql_safe_updates;

SELECT
    DATABASE() AS current_database,
    'seed_non_user_product_inventory_500' AS seed_name,
    (SELECT COUNT(*) FROM products WHERE description LIKE '[seed-500]%') AS products,
    (SELECT COUNT(*)
       FROM product_inventories AS pi
       INNER JOIN products AS p ON p.id = pi.product_id
      WHERE p.description LIKE '[seed-500]%') AS product_inventories,
    (SELECT COUNT(*) FROM stock_logs WHERE remark LIKE '[seed-500]%') AS stock_logs,
    (SELECT COUNT(*) FROM products WHERE description LIKE '[seed-500]%')
      + (SELECT COUNT(*)
           FROM product_inventories AS pi
           INNER JOIN products AS p ON p.id = pi.product_id
          WHERE p.description LIKE '[seed-500]%')
      + (SELECT COUNT(*) FROM stock_logs WHERE remark LIKE '[seed-500]%') AS total_rows;

SELECT p.id,
       p.name,
       p.status,
       p.price_fen,
       pi.stock_quantity
FROM products AS p
INNER JOIN product_inventories AS pi ON pi.product_id = p.id
WHERE p.description LIKE '[seed-500]%'
ORDER BY p.id
LIMIT 10;
