-- 清理测试数据
-- 1. 删除测试用户的评分
DELETE FROM rating WHERE user_id IN (SELECT id FROM user WHERE username LIKE 'loadtest_%');

-- 2. 删除收藏夹项目
DELETE FROM collection_item WHERE collection_id IN (SELECT id FROM collection WHERE user_id IN (SELECT id FROM user WHERE username LIKE 'loadtest_%'));

-- 3. 删除收藏夹
DELETE FROM collection WHERE user_id IN (SELECT id FROM user WHERE username LIKE 'loadtest_%');

-- 4. 删除测试用户
DELETE FROM user WHERE username LIKE 'loadtest_%';

-- 5. 删除评分事件（image_event）
DELETE FROM image_event WHERE user_id NOT IN (SELECT id FROM user);
