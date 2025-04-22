--解决并发问题
--验证码在redis上的key
local key=KEY[1] --验证码的key
local cntKey=key..":cnt" --验证码的key+cnt
local val=ARGV[1] --验证码的值
local ttl=tonumber(redis.call("ttl",key))  --验证码的过期时间

if ttl==-1 then --键已设置但没有过期时间
    return -2
elseif ttl==-2 or ttl<540 then --键不存在或过期时间小于540秒
    redis.call("set", key, val)
    redis.call("expire", key, 600) 
    redis.call("set", cntKey, 3)
    redis.call("expire", cntKey, 600)
else --键存在且过期时间大于540秒
    return -1    