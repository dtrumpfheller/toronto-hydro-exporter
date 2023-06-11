package torontohydro

import (
	"log"
	"net/http"
	"time"
)

func Mock() {
	log.Println("Mocking Toronto Hydro!")

	http.HandleFunc("/log-in", login)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/my-usage", myusage)

	go func() {
		log.Fatal(http.ListenAndServe(":9999", nil))
	}()
}

func login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<form action=\"https://www.torontohydro.com/log-in?p_p_id=th_module_authentication_ThModuleAuthenticationPortlet&p_p_lifecycle=1&p_p_state=normal&p_p_mode=view&_th_module_authentication_ThModuleAuthenticationPortlet_javax.portlet.action=%2Flogin&p_auth=PlXHUFya\" id=\"_th_module_authentication_ThModuleAuthenticationPortlet_authentication\" method=\"post\"></form>"))
	case "POST":
		w.Header().Set("Content-Type", "application/text")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Logged in!"))
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/text")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Logged out!"))
}

func myusage(w http.ResponseWriter, r *http.Request) {
	resource := r.URL.Query().Get("p_p_resource_id")
	if resource == "fetchMeterList" {
		data := `[
			{
				"endDate": "` + time.Now().Format("2006-01-02") + `",
				"meterNum": "1234",
				"id": "4321",
				"startDate": "` + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + `"
			}
		]`
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(data))
		return

	} else if resource == "getHourlyChartData" {
		data := `2020/01/01 10:10:00 # Your hourly usage (2020-01-01)
Time,Usage TOU off-peak (kWh),Usage TOU mid-peak (kWh),Usage TOU on-peak (kWh),Usage tier 1 (kWh),Usage tier 2 (kWh),Usage ULO overnight (kWh),Usage ULO off-peak (kWh),Usage ULO mid-peak (kWh),Usage ULO on-peak (kWh),Cost TOU off-peak ($),Cost TOU mid-peak ($),Cost TOU on-peak ($),Cost tier 1 ($),Cost tier 2 ($),Cost ULO overnight ($),Cost ULO off-peak ($),Cost ULO mid-peak ($),Cost ULO on-peak ($)
12 a.m.,,,,0.21,0.00,,,,,,,,0.02,0.00,,,,
1 a.m.,,,,0.21,0.00,,,,,,,,0.02,0.00,,,,
2 a.m.,,,,0.23,0.00,,,,,,,,0.02,0.00,,,,
3 a.m.,,,,0.17,0.00,,,,,,,,0.01,0.00,,,,
4 a.m.,,,,0.27,0.00,,,,,,,,0.02,0.00,,,,
5 a.m.,,,,0.17,0.00,,,,,,,,0.01,0.00,,,,
6 a.m.,,,,0.17,0.00,,,,,,,,0.01,0.00,,,,
7 a.m.,,,,0.34,0.00,,,,,,,,0.03,0.00,,,,
8 a.m.,,,,0.42,0.00,,,,,,,,0.04,0.00,,,,
9 a.m.,,,,0.34,0.00,,,,,,,,0.03,0.00,,,,
10 a.m.,,,,0.30,0.00,,,,,,,,0.03,0.00,,,,
11 a.m.,,,,1.07,0.00,,,,,,,,0.09,0.00,,,,
12 p.m.,,,,0.38,0.00,,,,,,,,0.03,0.00,,,,
1 p.m.,,,,0.38,0.00,,,,,,,,0.03,0.00,,,,
2 p.m.,,,,0.34,0.00,,,,,,,,0.03,0.00,,,,
3 p.m.,,,,0.38,0.00,,,,,,,,0.03,0.00,,,,
4 p.m.,,,,0.39,0.00,,,,,,,,0.03,0.00,,,,
5 p.m.,,,,0.50,0.00,,,,,,,,0.04,0.00,,,,
6 p.m.,,,,0.38,0.00,,,,,,,,0.03,0.00,,,,
7 p.m.,,,,0.31,0.00,,,,,,,,0.03,0.00,,,,
8 p.m.,,,,0.16,0.00,,,,,,,,0.01,0.00,,,,
9 p.m.,,,,0.25,0.00,,,,,,,,0.02,0.00,,,,
10 p.m.,,,,0.29,0.00,,,,,,,,0.03,0.00,,,,
11 p.m.,,,,0.16,0.00,,,,,,,,0.01,0.00,,,,`
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/text")
		w.Write([]byte(data))
		return
	}

	w.WriteHeader(http.StatusBadRequest)
}
