package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
)

// Reward endpoints
func (h *APIHandler) handleRewards(w http.ResponseWriter, r *http.Request) {
	h.authed(func(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext) {
		switch r.Method {
		case http.MethodPost:
			h.createReward(w, r, authCtx)
		default:
			h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		}
	})(w, r)
}

func (h *APIHandler) handleRewardsByFamily(w http.ResponseWriter, r *http.Request) {
	h.authed(func(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext) {
		familyUID := strings.TrimPrefix(r.URL.Path, "/api/v1/rewards/")
		if familyUID == "" {
			h.respondError(w, http.StatusBadRequest, ErrFamilyUIDRequired)
			return
		}

		parts := strings.SplitN(familyUID, "/", 2)
		familyUID = parts[0]

		if len(parts) == 2 && parts[1] != "" {
			rewardName, err := url.QueryUnescape(parts[1])
			if err != nil {
				h.respondError(w, http.StatusBadRequest, ErrInvalidEntityEncoding)
				return
			}

			switch r.Method {
			case http.MethodGet:
				h.getReward(w, r, authCtx, familyUID, rewardName)
			case http.MethodPut:
				h.updateReward(w, r, authCtx, familyUID, rewardName)
			case http.MethodDelete:
				h.deleteReward(w, r, authCtx, familyUID, rewardName)
			default:
				h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
			}
		} else {
			h.getFamilyRewards(w, r, authCtx, familyUID)
		}
	})(w, r)
}

func (h *APIHandler) createReward(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext) {
	var reward models.Rewards
	if err := h.decodeJSON(w, r, &reward); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	if validationErrors := h.ValidateRewardCreate(&reward); len(validationErrors) > 0 {
		h.respondError(w, http.StatusBadRequest, FormatValidationErrors(validationErrors))
		return
	}

	err := h.services.RewardService.CreateReward(r.Context(), authCtx, &reward)
	if err != nil {
		h.respondServiceError(w, err, "failed to create reward")
		return
	}

	h.respondSuccess(w, http.StatusCreated, reward)
}

func (h *APIHandler) getFamilyRewards(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, familyUID string) {
	rewards, err := h.services.RewardService.GetRewardsForFamily(r.Context(), authCtx, familyUID)
	if err != nil {
		h.respondServiceError(w, err, "failed to retrieve rewards")
		return
	}

	h.respondSuccess(w, http.StatusOK, rewards)
}

func (h *APIHandler) getReward(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, familyUID, rewardName string) {
	reward, err := h.services.RewardService.GetReward(r.Context(), authCtx, familyUID, rewardName)
	if err != nil {
		h.respondServiceError(w, err, "failed to retrieve reward")
		return
	}

	h.respondSuccess(w, http.StatusOK, reward)
}

func (h *APIHandler) updateReward(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, familyUID, rewardName string) {
	var updates domain.UpdateRewardRequest
	if err := h.decodeJSON(w, r, &updates); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	reward, err := h.services.RewardService.UpdateReward(r.Context(), authCtx, familyUID, rewardName, &updates)
	if err != nil {
		h.respondServiceError(w, err, "failed to update reward")
		return
	}

	h.respondSuccess(w, http.StatusOK, reward)
}

func (h *APIHandler) deleteReward(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, familyUID, rewardName string) {
	err := h.services.RewardService.DeleteReward(r.Context(), authCtx, familyUID, rewardName)
	if err != nil {
		h.respondServiceError(w, err, "failed to delete reward")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
