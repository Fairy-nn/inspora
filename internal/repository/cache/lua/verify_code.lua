local key = KEYS[1]
local expectedCode = ARGV[1]
local cnt = tonumber(redis.call("get", key.."cnt"))
local code = redis.call("get", key)

if cnt == nil or cnt <= 0 then
    return -1   -- user has used all attempts
elseif code == nil then
    return -2   -- code does not exist
elseif expectedCode == code then
    redis.call("del", key.."cnt") -- delete the count key
    redis.call("del", key) -- delete the code key
    return 0   -- code is correct
else
    redis.call("decr", key.."cnt") -- decrement the count
    return -2   -- code is incorrect
end