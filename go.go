package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/xuri/excelize/v2"
)

type points struct {
	Meta struct {
		APIVersion string `json:"api_version"`
		Code       int    `json:"code"`
		IssueDate  string `json:"issue_date"`
	} `json:"meta"`
	Result struct {
		Items []struct {
			AddressName string `json:"address_name,omitempty"`
			FullName    string `json:"full_name"`
			Geometry    struct {
				Centroid string `json:"centroid"`
			} `json:"geometry"`
			ID    string `json:"id"`
			Name  string `json:"name"`
			Point struct {
				Lat float64 `json:"lat"`
				Lon float64 `json:"lon"`
			} `json:"point"`
			PurposeName string `json:"purpose_name,omitempty"`
			Type        string `json:"type"`
		} `json:"items"`
		Total int `json:"total"`
	} `json:"result"`
}
type Agent struct {
	AgentID         int `json:"agent_id"`
	StartWaypointID int `json:"start_waypoint_id"`
}

type Point struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Waypoint struct {
	WaypointID int   `json:"waypoint_id"`
	Point      Point `json:"point"`
}

type RequestData struct {
	Agents    []Agent    `json:"agents"`
	Waypoints []Waypoint `json:"waypoints"`
}

type Task struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
	Urls   struct {
		URLVrpSolution string `json:"url_vrp_solution"`
		URLExcluded    string `json:"url_excluded"`
	} `json:"urls"`
	DmQueue  int `json:"dm_queue"`
	Dm       int `json:"dm"`
	Vrp      int `json:"vrp"`
	VrpQueue int `json:"vrp_queue"`
}
type P struct {
	Id     int
	Street string
	Lat    float64
	Lon    float64
}

type Way struct {
	Routes []struct {
		AgentID   int   `json:"agent_id"`
		Points    []int `json:"points"`
		Duration  int   `json:"duration"`
		Distance  int   `json:"distance"`
		Waypoints []struct {
			WaypointID         int `json:"waypoint_id"`
			DurationWaypoint   int `json:"duration_waypoint"`
			DistanceToWaypoint int `json:"distance_to_waypoint"`
		} `json:"waypoints"`
	} `json:"routes"`
	SummaryDuration int `json:"summary_duration"`
	SummaryDistance int `json:"summary_distance"`
}

func main() {
	poi, err := main1()
	if err != nil {
		fmt.Println(err)
		return
	}

	task := Task{}

	tic := time.NewTicker(10 * time.Second)
	for range tic.C {
		tk, err := main2(poi)
		task = tk
		if err != nil {
			fmt.Println(err)
			return
		}
		break
	}

	ticker := time.NewTicker(15 * time.Second)
	ts := Task{}
	for range ticker.C {
		tsk, err := main3(task)
		if err != nil {
			fmt.Println(err)
			ticker.Stop()
			return
		}
		if tsk.Status == "Done" {
			ts = tsk
			ticker.Stop()
			break
		}
	}
	err1 := main4(ts, poi)
	if err1 != nil {
		fmt.Println(err)
		return
	}

}

func main1() ([]P, error) {
	f, err := excelize.OpenFile("1.xlsx")
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	// Получить все строки в Sheet1
	rows, err := f.GetRows("Лист1")
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	var s []string
	for _, row := range rows {
		for _, colCell := range row {
			s = append(s, colCell)
		}
	}

	s = append(s[:0], s[1:]...)

	fmt.Println(s)
	// var pi []P это тоже самое, что и снизу
	pi := make([]P, 0, 10)
	ticker := time.NewTicker(2 * time.Second)
	for i, value := range s {
		for range ticker.C {
			t, _ := url.Parse("https://catalog.api.2gis.com/3.0/items/geocode")
			param := url.Values{}
			param.Add("q", value)
			param.Add("fields", "items.point,items.geometry.centroid")
			param.Add("city_id", "141360258613345")
			param.Add("key", "1fce38fa-1abc-4781-91d5-020b91a4d2e2")
			t.RawQuery = param.Encode()

			req, err := http.NewRequest("GET", t.String(), nil)

			if err != nil {
				fmt.Println(err)
			}
			client := &http.Client{}
			res, err := client.Do(req)
			if err != nil {
				panic(err)
			}
			defer res.Body.Close()

			if res.StatusCode != 200 {
				fmt.Println("Ошибка, статус запроса:", res.StatusCode, ". Программа закроется через 10 сек")
				ticker := time.NewTicker(10 * time.Second)
				for range ticker.C {
					os.Exit(0)
				}
			} else {
				fmt.Println(res.StatusCode)
			}

			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				fmt.Println("Ошибка чтения ответа:", err)
				return nil, err
			}

			Points := points{}
			err1 := json.Unmarshal(body, &Points)
			if err1 != nil {
				fmt.Println("Ошибка преобразования:", err1)
				return nil, err
			}
			pi = append(pi, P{
				Id:     i,
				Street: value,
				Lat:    Points.Result.Items[0].Point.Lat,
				Lon:    Points.Result.Items[0].Point.Lon,
			})
			break
		}

	}

	return pi, nil
}
func main2(p []P) (Task, error) {
	httpPostUrl := "https://routing.api.2gis.com/logistics/vrp/1.1.0/create?key=1fce38fa-1abc-4781-91d5-020b91a4d2e2"

	w := make([]Waypoint, 0, len(p))

	for i, v := range p {
		w = append(w, Waypoint{
			WaypointID: i,
			Point: Point{
				Lat: v.Lat,
				Lon: v.Lon,
			},
		})
	}

	requestData := RequestData{
		Agents: []Agent{
			{
				AgentID:         0,
				StartWaypointID: 0,
			},
		},
		Waypoints: w,
	}
	jsonBytes, err := json.Marshal(requestData)
	if err != nil {
		fmt.Println("Error:", err)
		return Task{}, err
	}

	req, err := http.NewRequest("POST", httpPostUrl, bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		fmt.Println(err)
		return Task{}, err
	}
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 201 {
		fmt.Println("Ошибка, статус запроса:", res.StatusCode, ". Программа закроется через 10 сек")
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			os.Exit(0)
		}
	} else {
		fmt.Println(res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Ошибка чтения ответа:", err)
		return Task{}, err
	}

	id := Task{}

	err1 := json.Unmarshal(body, &id)
	if err1 != nil {
		fmt.Println("Ошибка преобразования:", err1)
		return Task{}, err
	}

	return id, nil
}

func main3(t Task) (Task, error) {
	h, _ := url.Parse("https://routing.api.2gis.com/logistics/vrp/1.1.0/status")
	param := url.Values{}
	param.Add("task_id", t.TaskID)
	param.Add("key", "1fce38fa-1abc-4781-91d5-020b91a4d2e2")
	h.RawQuery = param.Encode()

	req, err := http.NewRequest("GET", h.String(), nil)
	if err != nil {
		fmt.Println(err)
	}
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		fmt.Println("Ошибка, статус запроса:", res.StatusCode, ". Программа закроется через 10 сек")
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			os.Exit(0)
		}
	} else {
		fmt.Println(res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Ошибка чтения ответа:", err)
		return Task{}, err
	}
	tsk := Task{}
	err1 := json.Unmarshal(body, &tsk)
	if err1 != nil {
		fmt.Println("Ошибка преобразования:", err1)
		return Task{}, err
	}

	return tsk, nil
}

func main4(t Task, p []P) error {
	h, _ := url.Parse(t.Urls.URLVrpSolution)
	param := url.Values{}
	h.RawQuery = param.Encode()
	req, err := http.NewRequest("GET", h.String(), nil)
	if err != nil {
		fmt.Println(err)
	}
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		fmt.Println("Ошибка, статус запроса:", res.StatusCode, "Программа закроется через 10 сек")
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			os.Exit(0)
		}
	} else {
		fmt.Println(res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Ошибка чтения ответа:", err)
		return err
	}
	way := Way{}
	err1 := json.Unmarshal(body, &way)
	if err1 != nil {
		fmt.Println("Ошибка преобразования:", err1)
		return err
	}

	file, err := os.Create("Маршрут.txt")
	if err != nil {
		fmt.Println("Ошибка создания файла:", err)
		return err
	}

	for _, v := range way.Routes[0].Points {
		for _, v1 := range p {
			if v1.Id == v {
				_, err = file.WriteString(v1.Street + " - ")
				if err != nil {
					fmt.Println("Ошибка записи в файл:", err)
					return err
				}
			}
		}
	}

	err = file.Close()
	if err != nil {
		fmt.Println("Ошибка закрытия файла:", err)
	}

	return nil

}
