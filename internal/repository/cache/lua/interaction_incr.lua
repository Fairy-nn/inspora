local key = KEYS[1] -- 对应的是hincrby的field
local cntKey = ARGV[1] -- 加一或者减一
local delta = tonumber(ARGV[2])
local exists = redis.call('EXISTS', key)

if exists == 1 then
    redis.call('HINCRBY', key, cntKey, delta)
    return 1
else
    return 0 -- 自增不成功
end