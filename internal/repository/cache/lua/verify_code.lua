-- 验证码验证
local key=KEY[1]
local expectedCode=ARGV[1]
local cnt =tonumber(redis.call("get", key..":cnt")) --获取验证码的次数
local code=redis.call("get", key) --获取验证码的值
if cnt == nil or cnt <= 0 then
    return -1  --验证码已失效
elseif code == nil then
    return -2 --验证码已过期
elseif expectedCode == code then
    redis.call("del", key.."cnt")  --删除验证码的次数
    redis.call("del", key)  --删除验证码的值
    return 0 --验证码正确
else
    redis.call("decr", key..":cnt") --减少验证码的次数
    return -2 --验证码错误
end