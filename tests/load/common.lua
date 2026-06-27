-- wrk 压测共通脚本
wrk.method = "GET"

-- 登录获取 token
local function login(username, password)
  local body = string.format('{"username":"%s","password":"%s"}', username, password)
  local response, status, _ = wrk.format("POST", "/api/v1/users/login", {}, nil, body)
  if status ~= 200 then
    print("Login failed for user " .. username)
    return nil
  end
  local res = wrk.responseBody
  local token = string.match(res, '"token":"([^"]+)"')
  return token
end

-- 切换线程初始化
local threadGroup = {}

wrk.thread = function(stage)
  local user = threadGroup[stage]
  if not user then
    return nil
  end
  wrk.headers["Authorization"] = "Bearer " .. user.token
  return user
end

wrk.init = function()
  threadGroup = {}
end
