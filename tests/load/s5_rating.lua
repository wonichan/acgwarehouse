-- S5: 评分压测 (写事务并发测试)
-- 并发评分同一张图片，测试数据一致性

local auth_header = nil
local image_id = 1  -- 固定评分同一张图片

function setup()
  -- 多用户登录
  local body = '{"username":"loadtest_user1","password":"Test123456"}'
  local response, status = wrk.format("POST", "/api/v1/users/login", {}, nil, body)
  if status == 200 then
    local res = wrk.responseBody
    auth_header = "Bearer " .. string.match(res, '"token":"([^"]+)"')
  end
end

function request()
  local score = math.random(0, 100)
  local body = '{"score":' .. score .. '}'
  local headers = {"Authorization: " .. auth_header}
  return wrk.format("PUT", "/api/v1/images/" .. image_id .. "/rating", headers, nil, body)
end
