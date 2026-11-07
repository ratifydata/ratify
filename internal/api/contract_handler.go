package api

import (
	"encoding/json"
	"net/http"

	"github.com/ratifydata/ratify/internal/contract"
)

func createContract(c *contract.ContractApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var params contract.ContractParams
		if json.NewDecoder(r.Body).Decode(&params) != nil {
			writeJSONResponse(w, http.StatusBadRequest, Response{Status: "error", Message: "invalid request body"})
			return
		}

		res, err := c.InitContract(r.Context(), params)
		if err != nil {
			writeHTTPError(w, err)
			return
		}

		writeJSONResponse(w, http.StatusCreated, Response{Status: "ok", Body: res})
	}
}

func activateContract(c *contract.ContractApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var params contract.ActivateContract
		if json.NewDecoder(r.Body).Decode(&params) != nil {
			writeJSONResponse(w, http.StatusBadRequest, Response{Status: "error", Message: "invalid request body"})
			return
		}
		err := c.ActivateContract(r.Context(), params)
		if err != nil {
			writeHTTPError(w, err)
			return
		}
		writeJSONResponse(w, http.StatusCreated, Response{Status: "ok"})
	}
}

func getContract(c *contract.ContractApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, valid := pathID(w, r, "id")
		if !valid {
			return
		}
		res, err := c.GetContract(r.Context(), id)
		if err != nil {
			writeHTTPError(w, err)
			return
		}
		writeJSONResponse(w, http.StatusOK, Response{Status: "ok", Body: res})
	}
}

func listContracts(c *contract.ContractApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contracts, err := c.ListContracts(r.Context(), r.URL.Query().Get("status"))
		if err != nil {
			writeHTTPError(w, err)
			return
		}
		writeJSONResponse(w, http.StatusOK, Response{Status: "ok", Body: contracts})
	}
}
