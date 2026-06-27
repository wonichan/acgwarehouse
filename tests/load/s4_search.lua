-- S4: 搜索压测 (Bleve 并发测试)
local keywords = {"test", "anime", "image", "art", "gallery"}

function request()
  local q = keywords[math.random(1, #keywords)]
  local page = math.random(1, 10)
  return wrk.format("GET", "/api/v1/search?q=" .. q .. "&page=" .. page .. "&size=20")
end
