package GuavaCache

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

type TestA struct {
	Value string
}

var g *LoadingCache

func init() {
	g = BuildLoadingCache(
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
		}))

}

func BenchmarkWithExpireAfterWrite(b *testing.B) {
	for i := 0; i < b.N; i++ {
		g.Get(rand.Intn(1000))
	}
}

func TestLRU(t *testing.T) {
	g := BuildLoadingCache(
		WithExpireAfterWrite(2*time.Second),
		WithMaximumSize(2),
		WithExpireAfterAccess(2*time.Second),
		WithLoader(func(key Key) (value Value, e error) {
			t := time.Now().UnixNano()
			return fmt.Sprintf("%d", t), nil
		}))
	g.Get(1)
	g.Get(2)
	g.Get(3)
	g.printLruCache()
	fmt.Println("-----")

	g.Get(2)
	time.Sleep(1 * time.Second)
	g.printLruCache()

	st := Stats{}
	g.Stats(&st)
	fmt.Printf("%+v\n", st)
}

func TestPut(t *testing.T) {
	g := BuildLoadingCache(WithLoader(func(key Key) (value Value, e error) {
		return TestA{Value: "abc"}, nil
	}),
		WithExpireAfterWrite(2*time.Second),
		WithMaximumSize(2),
		WithExpireAfterAccess(2*time.Second))
	g.Get(1)
	g.Get(2)
	g.Get(3)
	g.printLruCache()
	fmt.Println("-----")

	g.Put("张三", "谭柳")
	g.Get(2)
	time.Sleep(1 * time.Second)
	g.printLruCache()

	st := Stats{}
	g.Stats(&st)
	fmt.Printf("%+v\n", st)
}
func TestMinTime(t *testing.T) {

}

func TestGet(t *testing.T) {
	g := BuildLoadingCache(WithLoader(func(key Key) (value Value, e error) {
		fmt.Println("在print")
		return TestA{Value: key.(string)}, nil
	}),
		WithExpireAfterWrite(2*time.Second),
		WithMaximumSize(2),
		WithExpireAfterAccess(2*time.Second))
	key := "abc"
	for {
		g.Get(key)
		time.Sleep(1 * time.Second)
		//g.Remove("abc")
	}

}

func TestLoadingCache_GetWithExpiredFunc(t *testing.T) {

	type args struct {
		f CustomExpire
	}
	tests := []struct {
		name     string
		args     args
		wantSame bool
	}{
		{
			name:     "没有自定义方法",
			args:     args{f: nil},
			wantSame: true,
		},
		{
			name: "自定义了方法",
			args: args{f: func(value Value) bool {
				return true
			}},
			wantSame: false,
		},
	}

	flag := true
	g := BuildLoadingCache(WithLoader(func(key Key) (value Value, e error) {
		if flag == true {
			value = "start"
		}
		time.Sleep(1 * time.Second)

		return time.Now().Unix(), nil
	}),
		WithExpireAfterWrite(2*time.Hour),
		WithMaximumSize(2),
		WithExpireAfterAccess(2*time.Hour))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duplicateMap := make(map[int64]bool)
			for i := 0; i < 10; i++ {
				v, _ := g.GetWithExpiredFunc("abc", tt.args.f)
				duplicateMap[v.(int64)] = true
			}
			same := !(len(duplicateMap) == 10)
			if same != tt.wantSame {
				t.Errorf("error, want=%v,result=%v, len= %v", tt.wantSame, same, len(duplicateMap))
			}

		})
	}

}

func Test_getMinTimeDurationExcludeZero(t *testing.T) {
	type args struct {
		t1 time.Duration
		t2 time.Duration
	}
	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		{
			name: "test 1, 2",
			args: args{
				t1: 1 * time.Second,
				t2: 2 * time.Second,
			},
			want: 1 * time.Second,
		},
		{
			name: "test 2, 1",
			args: args{
				t1: 1 * time.Second,
				t2: 2 * time.Second,
			},
			want: 1 * time.Second,
		},
		{
			name: "test 0, 1",
			args: args{
				t1: 0 * time.Second,
				t2: 1 * time.Second,
			},
			want: 1 * time.Second,
		},
		{
			name: "test 1, 0",
			args: args{
				t1: 1 * time.Second,
				t2: 0 * time.Second,
			},
			want: 1 * time.Second,
		},
		{
			name: "test 0, 0",
			args: args{
				t1: 0 * time.Second,
				t2: 0 * time.Second,
			},
			want: 0 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getMinTimeDurationExcludeZero(tt.args.t1, tt.args.t2); got != tt.want {
				t.Errorf("getMinTimeDurationExcludeZero() = %v, want %v", got, tt.want)
			}
		})
	}
}
