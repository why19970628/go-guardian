-- 令牌桶限流器 (Token Bucket Rate Limiter)
-- KEYS[1]: 限流器唯一标识（例如：user:123:api_calls）
-- ARGV[1]: 请求令牌数（通常为 1）
-- ARGV[2]: 令牌生成速率（每秒生成的令牌数）
-- ARGV[3]: 桶容量（最大令牌数）

local key = KEYS[1]
local requested = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local capacity = tonumber(ARGV[3])

-- 获取当前时间（Redis 服务器时间）
local now = redis.call('TIME')
local nowInSeconds = tonumber(now[1])

-- 获取桶状态
local bucket = redis.call('HMGET', key, 'tokens', 'last_time')
local tokens = tonumber(bucket[1])
local last_time = tonumber(bucket[2])

-- 初始化桶（首次请求或过期）
if not tokens or not last_time then
    tokens = capacity
    last_time = nowInSeconds
else
    -- 计算新增令牌
    local elapsed = nowInSeconds - last_time
    local add_tokens = elapsed * rate
    tokens = math.min(capacity, tokens + add_tokens)
    last_time = nowInSeconds
end

-- 判断是否允许请求
local allowed = false
if tokens >= requested then
    tokens = tokens - requested
    allowed = true
end

-- 更新桶状态
redis.call('HMSET', key, 'tokens', tokens, 'last_time', last_time)

-- 返回结果（1 表示允许，0 表示拒绝）
return allowed and 1 or 0
