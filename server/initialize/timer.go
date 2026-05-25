package initialize

import (
	"context"
	"fmt"
	"github.com/flipped-aurora/gin-vue-admin/server/service"
	"github.com/flipped-aurora/gin-vue-admin/server/task"

	"github.com/robfig/cron/v3"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

func Timer() {
	go func() {
		var option []cron.Option
		option = append(option, cron.WithSeconds())
		// 清理DB定时任务
		_, err := global.GVA_Timer.AddTaskByFunc("ClearDB", "@daily", func() {
			err := task.ClearTable(global.GVA_DB) // 定时任务方法定在task文件包中
			if err != nil {
				fmt.Println("timer error:", err)
			}
		}, "定时清理数据库【日志，黑名单】内容", option...)
		if err != nil {
			fmt.Println("add timer error:", err)
		}

		// 其他定时任务定在这里 参考上方使用方法
		if global.GVA_CONFIG.Amazon.AutoSyncOrders {
			spec := global.GVA_CONFIG.Amazon.OrderSyncSpec
			if spec == "" {
				spec = "0 0 * * * *"
			}
			_, err = global.GVA_Timer.AddTaskByFunc("SyncAmazonOrders", spec, func() {
				if err := service.ServiceGroupApp.AmazonServiceGroup.OrderService.SyncEnabledStores(context.Background()); err != nil {
					fmt.Println("timer error:", err)
				}
			}, "定时同步 Amazon 新订单", option...)
			if err != nil {
				fmt.Println("add timer error:", err)
			}
		}

		listingSyncSpec := global.GVA_CONFIG.Amazon.ListingSyncSpec
		if listingSyncSpec == "" {
			listingSyncSpec = "0 */10 * * * *"
		}
		_, err = global.GVA_Timer.AddTaskByFunc("SyncAmazonListingSyncJobStatus", listingSyncSpec, func() {
			if err := service.ServiceGroupApp.AmazonServiceGroup.ListingSyncService.RefreshProcessingJobs(context.Background()); err != nil {
				fmt.Println("timer error:", err)
			}
		}, "定时同步 Amazon 价格库存回传任务状态", option...)
		if err != nil {
			fmt.Println("add timer error:", err)
		}

		fbaInventorySpec := global.GVA_CONFIG.Amazon.FBAInventorySyncSpec
		if fbaInventorySpec == "" {
			fbaInventorySpec = "0 30 * * * *"
		}
		_, err = global.GVA_Timer.AddTaskByFunc("SyncAmazonFBAInventory", fbaInventorySpec, func() {
			if err := service.ServiceGroupApp.AmazonServiceGroup.FBAInventorySyncService.SyncEnabledStores(context.Background()); err != nil {
				fmt.Println("timer error:", err)
			}
		}, "定时同步 Amazon FBA 实际库存", option...)
		if err != nil {
			fmt.Println("add timer error:", err)
		}

		returnSpec := global.GVA_CONFIG.Amazon.ReturnSyncSpec
		if returnSpec == "" {
			returnSpec = "0 15 * * * *"
		}
		_, err = global.GVA_Timer.AddTaskByFunc("SyncAmazonReturns", returnSpec, func() {
			if err := service.ServiceGroupApp.AmazonServiceGroup.ReturnService.SyncEnabledStores(context.Background()); err != nil {
				fmt.Println("timer error:", err)
			}
		}, "定时同步 Amazon 退货报表", option...)
		if err != nil {
			fmt.Println("add timer error:", err)
		}

		pickupSpec := global.GVA_CONFIG.Amazon.PickupSyncSpec
		if pickupSpec == "" {
			pickupSpec = "0 */10 * * * *"
		}
		_, err = global.GVA_Timer.AddTaskByFunc("SyncAmazonShipmentPickup", pickupSpec, func() {
			if err := service.ServiceGroupApp.AmazonServiceGroup.FulfillmentOrchestrator.RefreshPendingActualPickups(context.Background()); err != nil {
				fmt.Println("timer error:", err)
			}
		}, "定时同步 Amazon FBM 物流揽收时间", option...)
		if err != nil {
			fmt.Println("add timer error:", err)
		}

		shipmentConfirmRetrySpec := global.GVA_CONFIG.Amazon.ShipmentConfirmRetrySpec
		if shipmentConfirmRetrySpec == "" {
			shipmentConfirmRetrySpec = "0 */5 * * * *"
		}
		_, err = global.GVA_Timer.AddTaskByFunc("RetryAmazonShipmentConfirmation", shipmentConfirmRetrySpec, func() {
			if err := service.ServiceGroupApp.AmazonServiceGroup.ShipmentConfirmationService.RetryPendingConfirmations(context.Background()); err != nil {
				fmt.Println("timer error:", err)
			}
		}, "定时重试 Amazon 发货回传", option...)
		if err != nil {
			fmt.Println("add timer error:", err)
		}

		returnDispositionSpec := global.GVA_CONFIG.Amazon.ReturnDispositionSyncSpec
		if returnDispositionSpec == "" {
			returnDispositionSpec = "0 */15 * * * *"
		}
		_, err = global.GVA_Timer.AddTaskByFunc("SyncAmazonReturnDispositionStatus", returnDispositionSpec, func() {
			if err := service.ServiceGroupApp.AmazonServiceGroup.ReturnService.SyncPendingDispositions(context.Background()); err != nil {
				fmt.Println("timer error:", err)
			}
		}, "定时同步 Amazon 退货处置状态", option...)
		if err != nil {
			fmt.Println("add timer error:", err)
		}

		if global.GVA_CONFIG.Finance.Enabled {
			fxSpec := global.GVA_CONFIG.Finance.FXSyncSpec
			if fxSpec == "" {
				fxSpec = "0 0 6 * * *"
			}
			_, err = global.GVA_Timer.AddTaskByFunc("RefreshAmazonFinanceFXRates", fxSpec, func() {
				if err := service.ServiceGroupApp.AmazonServiceGroup.FinanceFXService.RefreshDailyRates(context.Background()); err != nil {
					fmt.Println("timer error:", err)
				}
			}, "定时刷新 Amazon 财务汇率", option...)
			if err != nil {
				fmt.Println("add timer error:", err)
			}

			settlementSpec := global.GVA_CONFIG.Finance.SettlementSyncSpec
			if settlementSpec == "" {
				settlementSpec = "0 10 6 * * *"
			}
			_, err = global.GVA_Timer.AddTaskByFunc("SyncAmazonSettlementReports", settlementSpec, func() {
				if err := service.ServiceGroupApp.AmazonServiceGroup.FinanceSettlementService.SyncSettlementReports(context.Background()); err != nil {
					fmt.Println("timer error:", err)
				}
			}, "定时同步 Amazon 财务结算报表", option...)
			if err != nil {
				fmt.Println("add timer error:", err)
			}

			adsSpec := global.GVA_CONFIG.Finance.AdsSyncSpec
			if adsSpec == "" {
				adsSpec = "0 20 6 * * *"
			}
			_, err = global.GVA_Timer.AddTaskByFunc("SyncAmazonAdsReports", adsSpec, func() {
				if err := service.ServiceGroupApp.AmazonServiceGroup.FinanceAdsService.SyncAdsReports(context.Background()); err != nil {
					fmt.Println("timer error:", err)
				}
			}, "定时同步 Amazon 财务广告报表", option...)
			if err != nil {
				fmt.Println("add timer error:", err)
			}

			recalcSpec := global.GVA_CONFIG.Finance.RecalcSpec
			if recalcSpec == "" {
				recalcSpec = "0 */10 * * * *"
			}
			_, err = global.GVA_Timer.AddTaskByFunc("ProcessAmazonFinanceRecalcJobs", recalcSpec, func() {
				if err := service.ServiceGroupApp.AmazonServiceGroup.FinanceRecalcService.ProcessPendingJobs(context.Background()); err != nil {
					fmt.Println("timer error:", err)
				}
			}, "定时处理 Amazon 财务回算任务", option...)
			if err != nil {
				fmt.Println("add timer error:", err)
			}
		}

		//_, err := global.GVA_Timer.AddTaskByFunc("定时任务标识", "corn表达式", func() {
		//	具体执行内容...
		//  ......
		//}, option...)
		//if err != nil {
		//	fmt.Println("add timer error:", err)
		//}
	}()
}
