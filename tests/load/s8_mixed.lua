-- S8: 混合场景压测 (模拟真实用户行为)
-- 读:写比例 约 8:2

local counters = {
  ping = 0,
  images = 0,
  image_detail = 0,
  search = 0,
  ratings = 0,
  rankings = 0,
  collections = 0
}

local image_ids = {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
local auth_header = nil
local logged_in = false

function setup()
  -- 登录测试用户获取 token
  local body = '{"username":"loadtest_user1","password":"Test123456"}'
  local response, status = wrk.format("POST", "/api/v1/users/login", {}, nil, body)
  if status == 200 then
    local res = wrk.responseBody
    auth_header = "Bearer " .. string.match(res, '"token":"([^"]+)"')
    logged_in = true
    print("Setup: logged in as loadtest_user1")
  else
    print("Setup: login failed, will use anonymous requests")
  end
end

function request()
  -- 按权重选择场景
  local weights = {
    {10, "ping"},
    {20, "images"},
    {15, "image_detail"},
    {15, "search"},
    {10, "ratings"},
    {10, "rankings"},
    {10, "collections"},
    {10, "rating_write"}
  }
  
  local total = 100
  local r = math.random(1, total)
  local cumulative = 0
  local choice = "ping"
  
  for _, w in ipairs(weights) do
    cumulative = cumulative + w[1]
    if r <= cumulative then
      choice = w[2]
      break
    end
  end
  
  counters[choice] = counters[choice] + 1
  
  local path
  local method = "GET"
  local headers = {}
  local body = nil
  
  if choice == "ping" then
    path = "/api/v1/ping"
    
  elseif choice == "images" then
    local page = math.random(1, 10)
    path = "/api/v1/images?page=" .. page .. "&size=20"
    
  elseif choice == "image_detail" then
    local id = image_ids[math.random(1, #image_ids)]
    path = "/api/v1/images/" .. id
    
  elseif choice == "search" then
    path = "/api/v1/search?q=test&page=1&size=20"
    
  elseif choice == "ratings" then
    local id = image_ids[math.random(1, #image_ids)]
    path = "/api/v1/images/" .. id .. "/rating"
    method = "PUT"
    headers = {"Authorization: " .. auth_header}
    body = '{"score":' .. math.random(0, 100) .. '}'
    
  elseif choice == "rankings" then
    local period = math.random(1, 3)
    local periods = {"day", "week", "month"}
    path = "/api/v1/rankings?period=" .. periods[period] .. "&page=1&size=20"
    
  elseif choice == "collections" then
    path = "/api/v1/collections"
    if logged_in then
      headers = {"Authorization: " .. auth_header}
    end
    
  elseif choice == "rating_write" then
    local id = image_ids[math.random(1, #image_ids)]
    path = "/api/v1/images/" .. id .. "/rating"
    method = "PUT"
    headers = {"Authorization: " .. auth_header}
    body = '{"score":' .. math.random(0, 100) .. '}'
  end
  
  return wrk.format(method, path, headers, nil, body)
end

function response(status, headers, body)
  -- 统计错误
  if status >= 400 then
    print("Error: status " .. status)
  end
end

function done(summary, latency, requests)
  print("\n=== Request Distribution ===")
  for k, v in pairs(counters) do
    print(k .. ": " .. v .. " requests")
  end
end
