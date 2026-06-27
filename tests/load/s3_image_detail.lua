-- S3: 图片详情压测 (ViewBuffer 并发测试)
local image_ids = {}
for i = 1, 200 do
  image_ids[i] = i
end

function request()
  local id = image_ids[math.random(1, #image_ids)]
  return wrk.format("GET", "/api/v1/images/" .. id)
end
