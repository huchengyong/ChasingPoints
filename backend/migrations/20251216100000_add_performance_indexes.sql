-- +goose Up
-- 添加复合索引优化查询性能

-- 用户+对手+状态复合索引，优化双向查询
CREATE INDEX idx_user_opponent_status ON matches(user_id, opponent_id, status);

-- 状态+删除时间复合索引，优化状态筛选
CREATE INDEX idx_status_deleted ON matches(status, deleted_at);

-- 对手ID+状态复合索引
CREATE INDEX idx_opponent_status ON matches(opponent_id, status);

-- +goose Down
DROP INDEX idx_user_opponent_status ON matches;
DROP INDEX idx_status_deleted ON matches;
DROP INDEX idx_opponent_status ON matches;
