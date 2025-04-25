local key = KEYS[1]
local expectedCode = ARGV[1]
local cnt = tonumber(redis.call("get", key.."cnt"))
local code = redis.call("get", key)

if cnt == nil or cnt <= 0 then
    return -1
elseif code == nil then
    return -2 
elseif expectedCode == code then
    redis.call("del", key.."cnt") 
    redis.call("del", key)
    return 0 
else
    redis.call("decr", key.."cnt") 
    return -2 
end