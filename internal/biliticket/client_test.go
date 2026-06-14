package biliticket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractProjectIDAcceptsSharedMallLinkWithTrailingPlus(t *testing.T) {
	link := "https://mall.bilibili.com/neul-next/ticket-renovation/detail.html?from_type=mall_home_search&id=1001701+&msource=mall_48111_ssk&outsideMall=no&outsideMall=no&plat_id=124&share_from=h5&share_medium=android&share_plat=android&share_session_id=ded4ebe7-6b84-4230-b000-67e9b9e1d628&share_source=COPY&share_tag=s_i&timestamp=1781318224&unique_k=gsBTdq4#themeType=1"

	projectID, err := ExtractProjectID(link)
	if err != nil {
		t.Fatalf("ExtractProjectID returned error: %v", err)
	}
	if projectID != 1001701 {
		t.Fatalf("projectID = %d, want 1001701", projectID)
	}
}

func TestExtractProjectIDAcceptsPaddedPlainID(t *testing.T) {
	projectID, err := ExtractProjectID(" 1001701+ ")
	if err != nil {
		t.Fatalf("ExtractProjectID returned error: %v", err)
	}
	if projectID != 1001701 {
		t.Fatalf("projectID = %d, want 1001701", projectID)
	}
}

func TestNormalizeBuyerAndAddress(t *testing.T) {
	buyer := normalizeBuyer(map[string]any{
		"id":          7,
		"name":        "张三",
		"personal_id": "110101199001010000",
		"phone":       "13800000000",
	})
	if buyer.ID != 7 || buyer.Name != "张三" || buyer.PersonalID == "" || buyer.Tel == "" {
		t.Fatalf("unexpected buyer: %#v", buyer)
	}

	address := normalizeAddress(map[string]any{
		"id":    9,
		"name":  "张三",
		"phone": "13800000000",
		"prov":  "上海市",
		"city":  "上海市",
		"area":  "徐汇区",
		"addr":  "测试路 1 号",
	})
	if address.ID != 9 || address.FullAddress != "上海市上海市徐汇区测试路 1 号" {
		t.Fatalf("unexpected address: %#v", address)
	}
}

func TestFetchPurchaseContextMapsProjectBuyersAndAddresses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/mall-search-items/items_detail/info":
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"data": map[string]any{
					"projectId":   1001701,
					"projectName": "测试项目",
					"hotProject":  false,
					"screenList": []map[string]any{
						{
							"id":          2001,
							"name":        "晚场",
							"start_time":  1781318224,
							"express_fee": 0,
							"ticket_list": []map[string]any{
								{
									"id":         3001,
									"desc":       "VIP",
									"price":      68000,
									"sale_start": "2026-06-13 20:00:00",
									"clickable":  true,
								},
							},
						},
					},
					"skuVenueInfo": map[string]any{
						"name":           "测试场馆",
						"address_detail": "测试地址",
					},
				},
			})
		case "/api/ticket/linkgoods/list":
			writeJSON(t, w, map[string]any{"code": 0, "data": map[string]any{"list": []any{}}})
		case "/x/web-interface/nav":
			writeJSON(t, w, map[string]any{"code": 0, "data": map[string]any{"uname": "测试用户"}})
		case "/api/ticket/buyer/list":
			writeJSON(t, w, map[string]any{
				"code": 0,
				"data": map[string]any{
					"list": []map[string]any{
						{"id": 7, "name": "张三", "personal_id": "110101199001010000", "phone": "13800000000"},
					},
				},
			})
		case "/api/ticket/addr/list":
			writeJSON(t, w, map[string]any{
				"code": 0,
				"data": map[string]any{
					"addr_list": []map[string]any{
						{"id": 9, "name": "张三", "phone": "13800000000", "prov": "上海市", "city": "上海市", "area": "徐汇区", "addr": "测试路 1 号"},
					},
				},
			})
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClientWithBaseURL(server.Client(), server.URL)
	project, err := client.FetchPurchaseContext(context.Background(), "1001701", "SESSDATA=test")
	if err != nil {
		t.Fatalf("FetchPurchaseContext: %v", err)
	}

	if project.ProjectID != 1001701 || project.ProjectName != "测试项目" {
		t.Fatalf("unexpected project: %#v", project)
	}
	if project.Username != "测试用户" {
		t.Fatalf("Username = %q, want 测试用户", project.Username)
	}
	if len(project.TicketOptions) != 1 || project.TicketOptions[0].SKUID != 3001 {
		t.Fatalf("unexpected ticket options: %#v", project.TicketOptions)
	}
	if len(project.Buyers) != 1 || project.Buyers[0].Name != "张三" {
		t.Fatalf("unexpected buyers: %#v", project.Buyers)
	}
	if len(project.Addresses) != 1 || project.Addresses[0].FullAddress != "上海市上海市徐汇区测试路 1 号" {
		t.Fatalf("unexpected addresses: %#v", project.Addresses)
	}
}

func TestFetchAccountContextDoesNotFetchProjectDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/x/web-interface/nav":
			writeJSON(t, w, map[string]any{"code": 0, "data": map[string]any{"uname": "测试用户"}})
		case "/api/ticket/buyer/list":
			writeJSON(t, w, map[string]any{
				"code": 0,
				"data": map[string]any{
					"list": []map[string]any{
						{"id": 7, "name": "张三", "personal_id": "110101199001010000"},
					},
				},
			})
		case "/api/ticket/addr/list":
			writeJSON(t, w, map[string]any{
				"code": 0,
				"data": map[string]any{
					"addr_list": []map[string]any{
						{"id": 9, "name": "张三", "phone": "13800000000", "prov": "上海市", "city": "上海市", "area": "徐汇区", "addr": "测试路 1 号"},
					},
				},
			})
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClientWithBaseURL(server.Client(), server.URL)
	context, err := client.FetchAccountContext(context.Background(), 1001701, "SESSDATA=test")
	if err != nil {
		t.Fatalf("FetchAccountContext: %v", err)
	}

	if context.ProjectID != 1001701 || context.Username != "测试用户" {
		t.Fatalf("unexpected context header: %#v", context)
	}
	if len(context.Buyers) != 1 || context.Buyers[0].Name != "张三" {
		t.Fatalf("unexpected buyers: %#v", context.Buyers)
	}
	if len(context.Addresses) != 1 || context.Addresses[0].ID != 9 {
		t.Fatalf("unexpected addresses: %#v", context.Addresses)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("Encode: %v", err)
	}
}
