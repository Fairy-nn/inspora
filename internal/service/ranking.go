package service

import (
	"context"
	"math"
	"time"

	"github.com/Fairy-nn/inspora/internal/domain"
	"github.com/Fairy-nn/inspora/internal/repository"
	"github.com/ecodeclub/ekit/queue"
)

type RankingServiceInterface interface {
	TopN(ctx context.Context) error
	GetTopN(ctx context.Context) ([]domain.Article, error)
}

type BatchRankService struct {
	artSvc    ArticleServiceInterface
	intrSvc   InteractionServiceInterface
	repo      repository.RankingRepositoryInterface
	batchsize int
	n         int
	scoreFunc func(t time.Time, likecnt int64) float64
}

func NewBatchRankService(artSvc ArticleServiceInterface, intrSvc InteractionServiceInterface, repo repository.RankingRepositoryInterface) RankingServiceInterface {
	return &BatchRankService{
		artSvc:    artSvc,
		intrSvc:   intrSvc,
		repo:      repo,
		batchsize: 100,
		n:         100,
		scoreFunc: func(t time.Time, likecnt int64) float64 {
			sec := time.Since(t).Seconds()
			return float64(likecnt-1) / math.Pow(float64(sec+2), 1.5)
		},
	}
}

// TopN 取出前N名的文章
func (b *BatchRankService) TopN(ctx context.Context) error {
	// 得到前n名的文章
	articles, err := b.topN(ctx)
	if err != nil {
		println("【排行榜】计算排名失败:", err.Error())
		return err
	}

	println("【排行榜】计算完成，文章数量:", len(articles))
	if len(articles) == 0 {
		println("【排行榜】没有找到合适的文章")
		// 避免缓存空结果
		return nil
	}

	// 将排名结果存储在redis缓存和本地缓存中
	err = b.repo.ReplaceTopN(ctx, articles)
	if err != nil {
		println("【排行榜】写入缓存失败:", err.Error())
	} else {
		println("【排行榜】写入缓存成功")
	}
	return err
}

// topN 获取前n名的文章
// 具体流程：分批获取最近7天的文章，获取每篇文章的交互信息（点赞数等），使用公式计算每篇文章的得分，维护一个优先队列，保留分数最高的 N 篇文章
func (b *BatchRankService) topN(ctx context.Context) ([]domain.Article, error) {
	// 只拿前7天的文章
	now := time.Now()
	offset := 0
	type Score struct {
		art   domain.Article
		score float64
	}

	//println("【排行榜计算】开始获取最近7天的文章...")

	// 使用队列来存储前n名的文章
	// 这里使用了一个并发安全的优先队列，分数高的优先级高
	topN := queue.NewConcurrentPriorityQueue[Score](b.n, func(a, b Score) int {
		if a.score == b.score {
			return 0
		} else if a.score > b.score {
			return 1
		} else {
			return -1
		}
	})

	// 记录总共处理的文章数
	totalArticles := 0
	articlesWithInteractions := 0

	// 分页获取文章及交互信息
	for {
		articles, err := b.artSvc.ListPublic(ctx, now, offset, b.batchsize)
		if err != nil {
			//println("【排行榜计算】获取文章列表失败:", err.Error())
			return nil, err
		}

		println("【排行榜计算】获取到文章", len(articles), "篇，偏移量:", offset)

		if len(articles) == 0 {
			//println("【排行榜计算】没有更多文章了")
			break
		}

		totalArticles += len(articles)

		// 文章ID列表
		ids := make([]int64, len(articles))
		for i := 0; i < len(articles); i++ {
			ids[i] = articles[i].ID
		}
		// 批量获取交互信息
		interactions, err := b.intrSvc.GetByIds(ctx, "article", ids)
		if err != nil {
			//println("【排行榜计算】获取文章交互信息失败:", err.Error())
			return nil, err
		}

		println("【排行榜计算】获取到交互信息", len(interactions), "条")

		// 文章分数计算与队列操作
		for _, article := range articles {
			intr, ok := interactions[article.ID]
			if !ok {
				continue
			}
			articlesWithInteractions++

			// 计算分数
			score := b.scoreFunc(now, intr.LikeCnt+2) //+2是为了避免分数为0

			err = topN.Enqueue(Score{
				art:   article,
				score: score,
			})
			// 如果队列满了，和分数最低的文章进行比较
			// 如果分数更高，就替换掉分数最低的文章
			if err == queue.ErrOutOfCapacity {
				val, _ := topN.Dequeue()
				if val.score < score {
					_ = topN.Enqueue(Score{
						art:   article,
						score: score,
					})
				} else {
					_ = topN.Enqueue(val)
				}
			}
		}

		// 如果没有更多的文章了，就退出循环
		// 这一批都没有取满，或者取到了7天前的文章，就退出循环
		if len(articles) < b.batchsize || now.Sub(articles[len(articles)-1].Utime) > 7*24*time.Hour {
			//println("【排行榜计算】达到终止条件，文章批次未满或已获取超过7天前的文章")
			break
		}

		offset += b.batchsize
	}

	println("【排行榜计算】总共处理文章:", totalArticles, "篇，有交互的文章:", articlesWithInteractions, "篇")

	res := make([]domain.Article, 0, b.n)
	for i := b.n - 1; i >= 0; i-- {
		val, err := topN.Dequeue()
		if err != nil {
			println("【排行榜计算】队列为空，已获取", len(res), "篇文章")
			break
		}
		res = append(res, val.art)
	}

	println("【排行榜计算】最终获取到排名文章", len(res), "篇")
	return res, nil
}

// GetTopN 获取前N名的文章
func (b *BatchRankService) GetTopN(ctx context.Context) ([]domain.Article, error) {
	articles, err := b.repo.GetTopN(ctx)
	if err != nil {
		return nil, err
	}
	return articles, nil
}
