// 无并发版本
package main

import (
    "bufio"
    "context"
    "fmt"
    "github.com/PuerkitoBio/goquery"
    "github.com/chromedp/cdproto/target"
    "github.com/chromedp/chromedp"
    "io"
    "log"
    "github.com/itchin/sq-music-download/model"
    "github.com/itchin/sq-music-download/util"
    "net/http"
    "net/http/httptest"
    "net/url"
    "os"
    "strconv"
    "strings"
)

func main() {
    ctx, cancel := chromedp.NewContext(context.Background())
    defer cancel()

    ch := addNewTabListener(ctx)

    var musicName string

    fmt.Print("输入要搜索的歌曲名称(按回车键结束): ")
    inputReader := bufio.NewReader(os.Stdin)
    musicName, err := inputReader.ReadString('\n')
    if err != nil {
        panic(err)
    }
    fmt.Println()

    searchMusic(ctx, musicName)
    ml := searchResultList(ctx, ch)

    linkUrl, quantity := selectMusic(ml)

    ctx, cancel = chromedp.NewContext(context.Background())
    defer cancel()

    //linkUrl := "https://music.migu.cn/v3/music/song/60060301610"
    //quantity := "HQ"

    ch = addNewTabListener(ctx)
    openPlayerTag(ctx, linkUrl)

    musicUrl := playMusic(ctx, ch, quantity)

    //ioutil.WriteFile("migu.txt", []byte(res), 0666)
    musicUrl = musicSrc(musicUrl)
    downlaodMusic(musicUrl)
    fmt.Println("下载完成，按回车键退出...")
    fmt.Scanln(&musicUrl)
}

/**
 * 下载歌曲
 */
func downlaodMusic(musicUrl string) {
    //musicUrl := "https://freetyst.nf.migu.cn/public/product12/2018/07/10/%E6%97%A0%E6%8D%9F/2017%E5%B9%B412%E6%9C%8822%E6%97%A516%E7%82%B934%E5%88%86%E5%86%85%E5%AE%B9%E5%87%86%E5%85%A5%E4%B8%AD%E5%94%B1%E8%89%BA%E8%83%BD%E9%A2%84%E7%95%99454%E9%A6%96/flac/%E5%8C%97%E4%BA%AC%E6%AC%A2%E8%BF%8E%E4%BD%A0-%E5%88%98%E7%B4%AB%E7%8E%B2.flac"
    // url解码
    musicUrl, err := url.QueryUnescape(musicUrl)
    if err != nil {
        panic(err)
    }
    index := strings.LastIndex(musicUrl, "/")
    musicName := musicUrl[index + 1:]
    resp, err := http.Get(musicUrl)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()
    length := resp.Header.Get("content-length")
    //fmt.Println("content-length:", length)

    file, err := os.Create(musicName)
    defer file.Close()
    if err != nil {
        panic(err)
    }
    //fmt.Println("开始下载...", musicUrl)

    // 方法一(荐)、使用io.Copy
    wt := bufio.NewWriter(file)
    //counter := new(util.WriteCounter)
    size, _ := strconv.ParseUint(length, 0, 64)
    counter := &util.WriteCounter{Size: size}
    counter.Init()
    var n int64
    //io.Copy回调方法实现实时计算下载进度
    if n, err = io.Copy(file, io.TeeReader(resp.Body, counter)); err != nil {
        file.Close()
        panic(err)
    }
    fmt.Println("write" , n)
    if err != nil {
        panic(err)
    }
    wt.Flush()

    // 方法二、使用字节流分段读写
    // TODO：存在问题：字节数总长度与http response的content-length是一致的，但写入到本地时字节大小产生波动。
    //buf := make([]byte, 8 * 1024)
    //size := 0
    //
    //for {
    //   n, err := resp.Body.Read(buf)
    //   size += n
    //   if err == io.EOF {
    //       fmt.Println("SUCCESS")
    //       break
    //   } else if err != nil {
    //       fmt.Println(err)
    //       break
    //   }
    //   file.Write(buf[0:n])
    //}
    //fmt.Println("size:", size)
}

/**
 * 打印输出结果
 */
func selectMusic(ml []*model.Music) (string, string) {
    //打印出歌曲列表
    fmt.Println("序号\t歌曲\t音质\t歌手\t专辑")
    i := 1
    for _, v := range ml {
        fmt.Println(i, v.Title, v.Quality, v.Singer, v.Album)
        i++
    }
    fmt.Println()

    var number int
    fmt.Print("输入数字，选择要下载的歌曲(按回车键结束): ")
    fmt.Scanln(&number)
    length := len(ml)
    if number > length {
        panic("超出所选长度")
    }

    m := ml[number - 1]
    return "https://music.migu.cn" + m.LinkUrl, m.Quality
}

/**
 * 触发搜索动作
 */
func searchMusic(ctx context.Context, musicName string)  {
    err := chromedp.Run(ctx,
        chromedp.Navigate("https://music.migu.cn/v3/music/player/audio"),
        chromedp.WaitVisible("#search_ipt", chromedp.ByID),
        chromedp.SetValue("#search_ipt", musicName, chromedp.ByID),
        chromedp.Click(`i[class="iconfont cf-nav-sousuo"]`, chromedp.BySearch),
    )
    if err != nil {
        log.Fatal(err)
    }
}

/**
 * 获取搜索结果页，返回切片指针
 */
func searchResultList(ctx context.Context, ch <-chan target.ID) []*model.Music {
    newCtx, cancel := chromedp.NewContext(ctx, chromedp.WithTargetID(<-ch))
    defer cancel()

    var res string
    err := chromedp.Run(newCtx,
        chromedp.OuterHTML(`div[class="songlist-body"]`, &res, chromedp.BySearch),
    )
    if err != nil {
        panic(err)
    }
    //ioutil.WriteFile("migu.txt", []byte(res), 0666)
    return searchPageParse(res)
}

/**
 * 解析搜索页html
 */
func searchPageParse(res string) (musics []*model.Music) {
    r := strings.NewReader(res)
    doc, err := goquery.NewDocumentFromReader(r)
    if err != nil {
        panic("读取html模板失败")
    }

    musics = make([]*model.Music, 0)
    doc.Find(".J-btn-share").Each(func(i int, selection *goquery.Selection) {
        data_share, _ := selection.Attr("data-share")
        //fmt.Println(data_share)
        m := new(model.Music)
        err := m.UnmarshalJSON([]byte(data_share))
        if err != nil {
            panic(err)
        }
        musics = append(musics, m)
    })
    doc.Find(".song-name-txt").Each(func(i int, selection *goquery.Selection) {
        m := musics[i]
        m.Quality = selection.Next().Text()
    })
    return
}
/**
 * 注册新tab标签的监听服务
 */
func addNewTabListener(ctx context.Context) <-chan target.ID {
    mux := http.NewServeMux()
    ts := httptest.NewServer(mux)
    defer ts.Close()

    return chromedp.WaitNewTarget(ctx, func(info *target.Info) bool {
        return info.URL != ""
    })
}

/**
 * 打开播放歌曲页面
 */
func openPlayerTag(ctx context.Context, url string)  {
    err := chromedp.Run(ctx,
        chromedp.Navigate(url),
        chromedp.WaitVisible(`#is_songPlay`, chromedp.ByID),
        chromedp.Click("#is_songPlay", chromedp.ByID),
    )
    if err != nil {
        log.Fatal(err)
    }
}

/**
 * 在新的容器中打开新的浏览器标签
 */
func playMusic(ctx context.Context, ch <-chan target.ID, quantity string) string {
    newCtx, cancel := chromedp.NewContext(ctx, chromedp.WithTargetID(<-ch))
    defer cancel()

    if quantity == "3D" {
        quantity = "D3"
    }
    var style,musicUrl string
    // 检查是否只有一种音质
    err := chromedp.Run(newCtx,
        chromedp.OuterHTML(`i[class="iconfont cf-shang"]`, &style, chromedp.BySearch),
        chromedp.OuterHTML("#migu_audio", &musicUrl, chromedp.ByID),
    )
    if err != nil {
        panic(err)
    }
    r := strings.NewReader(style)
    doc, _ := goquery.NewDocumentFromReader(r)
    s := doc.Find("i").First()
    style, _ = s.Attr("style")

    //如果有多种音质，切换为最高音质
    if style == "" {
        err = chromedp.Run(newCtx,
            chromedp.Click(`i[class="iconfont cf-shang"]`, chromedp.BySearch),              //点击，选择音质
            chromedp.WaitVisible(`i[class="iconfont cf-shang active"]`, chromedp.BySearch),//等待音质列表加载完成
            chromedp.Click(`span[class="` + quantity + `-rate"]`, chromedp.BySearch),        //点击，选择无损音质
            chromedp.WaitVisible(`b[class="` + quantity + `"]`, chromedp.BySearch),
            chromedp.OuterHTML("#migu_audio", &musicUrl, chromedp.ByID),
        )
        if err != nil {
            panic(err)
        }
    }

    return musicUrl
}

/**
 * 输入audio的html节点，截取歌曲的下载路径并返回
 */
func musicSrc(musicUrl string) string {
    r := strings.NewReader(musicUrl)
    doc, _ := goquery.NewDocumentFromReader(r)
    s := doc.Find("audio").First()
    src, bool := s.Attr("src")
    if bool == false {
        panic("scr属性不存在")
    }
    index := strings.Index(src, "?")
    return "http:" + src[:index]
}
