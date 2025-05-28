package protocols

import "sensor-edge/types"

// MapPointsToConfig 将 types.PointMapping 转换为 PointConfig
func MapPointsToConfig(points []types.PointMapping) []PointConfig {
	var configs []PointConfig
	for _, p := range points {
		configs = append(configs, PointConfig{
			PointID:   p.Name,
			Address:   p.Address,
			Type:      p.Type,
			Unit:      p.Unit,
			Transform: p.Transform,
		})
	}
	return configs
}
