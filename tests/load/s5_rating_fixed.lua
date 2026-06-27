-- S5: 评分压测 (写事务并发测试)
local auth_header = os.getenv("AUTH_HEADER") or ""
local image_id = 1

function request()
  local score = math.random(0, 100)
  local body = string.format('{"score":%d}', score)
  return wrk.format("PUT", "/api/v1/images/" .. image_id .. "/rating", 
    {"Content-Type: application/json", "Authorization: " .. auth_header}, nil, body)
end
