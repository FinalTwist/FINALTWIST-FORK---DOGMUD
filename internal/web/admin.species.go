package web

import (
	"net/http"
	"sort"
	"strconv"
	"text/template"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/species"
)

func speciesIndex(w http.ResponseWriter, r *http.Request) {

	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFiles(configs.GetFilePathsConfig().AdminHtml.String()+"/_header.html", configs.GetFilePathsConfig().AdminHtml.String()+"/species/index.html", configs.GetFilePathsConfig().AdminHtml.String()+"/_footer.html")
	if err != nil {
		mudlog.Error("HTML Template", "error", err)
	}

	allSpecies := species.GetAllSpecies()

	sort.SliceStable(allSpecies, func(i, j int) bool {
		return allSpecies[i].SpeciesId < allSpecies[j].SpeciesId
	})

	speciesIndexData := struct {
		Species []species.Species
	}{
		allSpecies,
	}

	if err := tmpl.Execute(w, speciesIndexData); err != nil {
		mudlog.Error("HTML Execute", "error", err)
	}

}

func speciesData(w http.ResponseWriter, r *http.Request) {

	tmpl, err := template.New("species.data.html").Funcs(funcMap).ParseFiles(configs.GetFilePathsConfig().AdminHtml.String() + "/species/species.data.html")
	if err != nil {
		mudlog.Error("HTML Template", "error", err)
	}

	urlVals := r.URL.Query()

	speciesIdInt, _ := strconv.Atoi(urlVals.Get(`speciesid`))

	speciesInfo := species.GetSpecies(speciesIdInt)
	if speciesInfo == nil {
		speciesInfo = &species.Species{}
	}

	tplData := map[string]any{}
	tplData[`speciesInfo`] = *speciesInfo

	conditionSpecs := []conditions.ConditionSpec{}
	for _, conditionId := range conditions.GetAllConditionIds() {
		if b := conditions.GetConditionSpec(conditionId); b != nil {
			if b.Name == `empty` {
				continue
			}
			conditionSpecs = append(conditionSpecs, *b)
		}
	}
	sort.SliceStable(conditionSpecs, func(i, j int) bool {
		return conditionSpecs[i].ConditionId < conditionSpecs[j].ConditionId
	})
	tplData[`buffSpecs`] = conditionSpecs

	tplData[`allSlotTypes`] = characters.GetAllSlotTypes()

	if err := tmpl.Execute(w, tplData); err != nil {
		mudlog.Error("HTML Execute", "error", err)
	}

}
