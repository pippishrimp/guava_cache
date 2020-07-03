# 背景
本地缓存需要具备几个功能
1. 缓存有限数据
2. 支持缓存过ttl
3. 支持缓存自动加载

# 功能
对于本地缓存的特性。guava支持了以下功能：
1. 支持缓存的容量配置。利用lru支持有限的数据
   - maximumSize
2. 支持3种ttl配置。
   - expireAfterAccess 多久不访问就会过期
   - expireAfterWrite  写数据后多久过期
   - refreshAfterWrite 数据多久后refresh,过期后第一个请求会触发异步更新缓存。但会返回旧的值，直到缓存被更新
3. 支持防击穿加载数据。
   - loader，当缓存中获取不到数据后，调用方法loader,防击穿。
4. 命中数据统计

# 使用方法
```
	g = GuavaCache.BuilderLoadingCache(
		//配置容量
		WithMaximumSize(1000),
		// 配置读超时
		WithExpireAfterAccess(40*time.Second),
		//配置写超时
		WithRefreshAfterWrite(30*time.Second),
		//配置加载的方法
		WithLoader(func(key Key) (value Value, e error) {
			t := time.Now().UnixNano()
			return fmt.Sprintf("%d", t), nil
		}),)
```
支持以下方法
- Get
- Put
- Remove
