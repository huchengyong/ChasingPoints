-- +goose Up
-- 修正 WPBSA 2024/25 斯诺克默认规则文案

UPDATE `rules_content`
SET `content` = '每局首杆母球为D区内手中球，红球为目标球；标准斯诺克不要求该杆必须有红球碰库或入袋。'
WHERE `category` = 'snooker' AND `content_type` = 'rule' AND `title` = '开球规则';

UPDATE `rules_content`
SET `content` = '若母球无法沿直线直接击中至少一颗目标球的两个极边，则形成斯诺克；仅遮挡部分线路不一定构成斯诺克。'
WHERE `category` = 'snooker' AND `content_type` = 'rule' AND `title` = '斯诺克规则';

UPDATE `rules_content`
SET `content` = '母球通常不得同时首碰两球；目标球为红球时同时首碰两颗红球合法，自由球情况下也可同时首碰指定自由球与目标球。'
WHERE `category` = 'snooker' AND `content_type` = 'foul' AND `title` = '同时击中两球';

UPDATE `rules_content`
SET `content` = '母球被非目标球阻挡，无法沿直线击中至少一颗目标球两个极边的局面。'
WHERE `category` = 'snooker' AND `content_type` = 'glossary' AND `title` = '斯诺克';

UPDATE `rules_content`
SET `content` = '无自由球的标准15红球局中，单杆最高为147分；若开局因对手犯规获得自由球，理论最高单杆可达155分。'
WHERE `category` = 'snooker' AND `content_type` = 'glossary' AND `title` = '满分147';

UPDATE `rules_content`
SET `content` = '对手犯规后，若下一击球方对全部目标球形成斯诺克，裁判宣告自由球；击球方可指定一颗非目标球取得目标球身份和分值。'
WHERE `category` = 'snooker' AND `content_type` = 'glossary' AND `title` = '自由球';

-- +goose Down

UPDATE `rules_content`
SET `content` = '首杆需从开球区内击打红球堆，至少有一颗红球碰库或入袋。'
WHERE `category` = 'snooker' AND `content_type` = 'rule' AND `title` = '开球规则';

UPDATE `rules_content`
SET `content` = '当目标球被阻挡无法直接击打时形成斯诺克，对手需尝试解球。'
WHERE `category` = 'snooker' AND `content_type` = 'rule' AND `title` = '斯诺克规则';

UPDATE `rules_content`
SET `content` = '母球首碰出现两颗目标球或非法连击，判犯规。'
WHERE `category` = 'snooker' AND `content_type` = 'foul' AND `title` = '同时击中两球';

UPDATE `rules_content`
SET `content` = '通过障碍球让对手无法直接击中目标球的防守局面。'
WHERE `category` = 'snooker' AND `content_type` = 'glossary' AND `title` = '斯诺克';

UPDATE `rules_content`
SET `content` = '标准规则下单杆可达到的最高分数。'
WHERE `category` = 'snooker' AND `content_type` = 'glossary' AND `title` = '满分147';

UPDATE `rules_content`
SET `content` = '对手犯规后形成斯诺克时可指定任意球代替目标球。'
WHERE `category` = 'snooker' AND `content_type` = 'glossary' AND `title` = '自由球';
