package hevc

import (
	"bytes"
	"fmt"

	"github.com/edgeware/mp4ff/bits"
)

// SPS - HEVC SPS parameters
// ISO/IEC 23008-2 Sec. 7.3.2.2
type SPS struct {
	VpsID                                byte
	MaxSubLayersMinus1                   byte
	TemporalIDNestingFlag                bool
	ProfileTierLevel                     ProfileTierLevel
	SpsID                                byte
	ChromaFormatIDC                      byte
	SeparateColourPlaneFlag              bool
	ConformanceWindowFlag                bool
	PicWidthInLumaSamples                uint32
	PicHeightInLumaSamples               uint32
	ConformanceWindow                    ConformanceWindow
	BitDepthLumaMinus8                   byte
	BitDepthChromaMinus8                 byte
	Log2MaxPicOrderCntLsbMinus4          byte
	SubLayerOrderingInfoPresentFlag      bool
	SubLayeringOrderingInfos             []SubLayerOrderingInfo
	Log2MinLumaCodingBlockSizeMinus3     byte
	Log2DiffMaxMinLumaCodingBlockSize    byte
	Log2MinLumaTransformBlockSizeMinus2  byte
	Log2DiffMaxMinLumaTransformBlockSize byte
	MaxTransformHierarchyDepthInter      byte
	MaxTransformHierarchyDepthIntra      byte
	ScalingListEnabledFlag               bool
	ScalingListDataPresentFlag           bool
	AmpEnabledFlag                       bool
	SampleAdaptiveOffsetEnabledFlag      bool
	PCMEnabledFlag                       bool
	NumShortTermRefPicSets               byte
	LongTermRefPicsPresentFlag           bool
	SpsTemporalMvpEnabledFlag            bool
	StrongIntraSmoothingEnabledFlag      bool
	VUIParametersPresentFlag             bool
	VUIParameters                        *VUIParameters
}

// ProfileTierLevel according to ISO/IEC 23008-2 Section 7.3.3
type ProfileTierLevel struct {
	GeneralProfileSpace              byte
	GeneralTierFlag                  bool
	GeneralProfileIDC                byte
	GeneralProfileCompatibilityFlags uint32
	GeneralConstraintIndicatorFlags  uint64 // 48 bits
	GeneralProgressiveSourceFlag     bool
	GeneralInterlacedSourceFlag      bool
	GeneralNonPackedConstraintFlag   bool
	GeneralFrameOnlyConstraintFlag   bool
	// 43 + 1 bits of info
	GeneralLevelIDC byte
	// Sublayer stuff

}

// ConformanceWindow according to ISO/IEC 23008-2
type ConformanceWindow struct {
	LeftOffset   uint32
	RightOffset  uint32
	TopOffset    uint32
	BottomOffset uint32
}

// SubLayerOrderingInfo according to ISO/IEC 23008-2
type SubLayerOrderingInfo struct {
	MaxDecPicBufferingMinus1 byte
	MaxNumReorderPics        byte
	MaxLatencyIncreasePlus1  byte
}

// ParseSPSNALUnit - Parse HEVC SPS NAL unit starting with NAL unit header
func ParseSPSNALUnit(data []byte) (*SPS, error) {

	sps := &SPS{}

	rd := bytes.NewReader(data)
	r := bits.NewAccErrEBSPReader(rd)
	// Note! First two bytes are NALU Header

	naluHdrBits := r.Read(16)
	naluType := GetNaluType(byte(naluHdrBits >> 8))
	if naluType != NALU_SPS {
		return nil, fmt.Errorf("NALU type is %s not SPS", naluType)
	}
	sps.VpsID = byte(r.Read(4))
	sps.MaxSubLayersMinus1 = byte(r.Read(3))
	sps.TemporalIDNestingFlag = r.ReadFlag()
	sps.ProfileTierLevel.GeneralProfileSpace = byte(r.Read(2))
	sps.ProfileTierLevel.GeneralTierFlag = r.ReadFlag()
	sps.ProfileTierLevel.GeneralProfileIDC = byte(r.Read(5))
	sps.ProfileTierLevel.GeneralProfileCompatibilityFlags = uint32(r.Read(32))
	sps.ProfileTierLevel.GeneralConstraintIndicatorFlags = uint64(r.Read(48))
	sps.ProfileTierLevel.GeneralLevelIDC = byte(r.Read(8))
	if sps.MaxSubLayersMinus1 != 0 {
		return sps, nil // Cannot parse any further
	}
	sps.SpsID = byte(r.ReadExpGolomb())
	sps.ChromaFormatIDC = byte(r.ReadExpGolomb())
	if sps.ChromaFormatIDC == 3 {
		sps.SeparateColourPlaneFlag = r.ReadFlag()
	}
	sps.PicWidthInLumaSamples = uint32(r.ReadExpGolomb())
	sps.PicHeightInLumaSamples = uint32(r.ReadExpGolomb())
	sps.ConformanceWindowFlag = r.ReadFlag()
	if sps.ConformanceWindowFlag {
		sps.ConformanceWindow = ConformanceWindow{
			LeftOffset:   uint32(r.ReadExpGolomb()),
			RightOffset:  uint32(r.ReadExpGolomb()),
			TopOffset:    uint32(r.ReadExpGolomb()),
			BottomOffset: uint32(r.ReadExpGolomb()),
		}
	}
	sps.BitDepthLumaMinus8 = byte(r.ReadExpGolomb())
	sps.BitDepthChromaMinus8 = byte(r.ReadExpGolomb())
	sps.Log2MaxPicOrderCntLsbMinus4 = byte(r.ReadExpGolomb())
	sps.SubLayerOrderingInfoPresentFlag = r.ReadFlag()
	startValue := byte(0)
	if sps.SubLayerOrderingInfoPresentFlag {
		startValue = sps.MaxSubLayersMinus1
	}
	for i := startValue; i <= sps.MaxSubLayersMinus1; i++ {
		sps.SubLayeringOrderingInfos = append(
			sps.SubLayeringOrderingInfos,
			SubLayerOrderingInfo{
				MaxDecPicBufferingMinus1: byte(r.ReadExpGolomb()),
				MaxNumReorderPics:        byte(r.ReadExpGolomb()),
				MaxLatencyIncreasePlus1:  byte(r.ReadExpGolomb()),
			})
	}
	sps.Log2MinLumaCodingBlockSizeMinus3 = byte(r.ReadExpGolomb())
	sps.Log2DiffMaxMinLumaCodingBlockSize = byte(r.ReadExpGolomb())
	sps.Log2MinLumaTransformBlockSizeMinus2 = byte(r.ReadExpGolomb())
	sps.Log2DiffMaxMinLumaTransformBlockSize = byte(r.ReadExpGolomb())
	sps.MaxTransformHierarchyDepthInter = byte(r.ReadExpGolomb())
	sps.MaxTransformHierarchyDepthIntra = byte(r.ReadExpGolomb())
	sps.ScalingListEnabledFlag = r.ReadFlag()
	if sps.ScalingListEnabledFlag {
		sps.ScalingListDataPresentFlag = r.ReadFlag()
		if sps.ScalingListDataPresentFlag {
			return sps, r.AccError() // Doesn't get any further now
		}
	}
	sps.AmpEnabledFlag = r.ReadFlag()
	sps.SampleAdaptiveOffsetEnabledFlag = r.ReadFlag()
	sps.PCMEnabledFlag = r.ReadFlag()
	if sps.PCMEnabledFlag {
		return sps, r.AccError() // Doesn't get any further now
	}
	sps.NumShortTermRefPicSets = byte(r.ReadExpGolomb())
	if sps.NumShortTermRefPicSets != 0 {
		return sps, r.AccError() // Doesn't get any further for now
	}
	sps.LongTermRefPicsPresentFlag = r.ReadFlag()
	if sps.LongTermRefPicsPresentFlag {
		return sps, r.AccError() // Does't get any further for now
	}
	sps.SpsTemporalMvpEnabledFlag = r.ReadFlag()
	sps.StrongIntraSmoothingEnabledFlag = r.ReadFlag()
	sps.VUIParametersPresentFlag = r.ReadFlag()
	if sps.VUIParametersPresentFlag {
		sps.VUIParameters = readVUIParameters(r)
	}

	return sps, r.AccError()
}

// ImageSize - calculated width and height using ConformanceWindow
func (s *SPS) ImageSize() (width, height uint32) {
	encWidth, encHeight := s.PicWidthInLumaSamples, s.PicHeightInLumaSamples
	var subWidthC, subHeightC uint32 = 1, 1
	switch s.ChromaFormatIDC {
	case 1: // 4:2:0
		subWidthC, subHeightC = 2, 2
	case 2: // 4:2:2
		subWidthC = 2
	}
	width = encWidth - (s.ConformanceWindow.LeftOffset+s.ConformanceWindow.RightOffset)*subWidthC
	height = encHeight - (s.ConformanceWindow.TopOffset+s.ConformanceWindow.BottomOffset)*subHeightC
	return width, height
}

// VUI - HEVC VUI parameters
// ISO/IEC 23008-2 Sec. E.2.1
type VUIParameters struct {
	AspectRatioInfoPresentFlag     bool
	AspectRatioIDC                 byte
	SARWidth                       uint16
	SARHeight                      uint16
	OverscanInfoPresentFlag        bool
	OverscanAppropriateFlag        bool
	VideoSignalTypePresentFlag     bool
	VideoFormat                    byte
	VideoFullRangeFlag             bool
	ColourDescriptionPresentFlag   bool
	ColourPrimaries                byte
	TransferCharacteristics        byte
	MatrixCoeffs                   byte
	ChromaLocInfoPresentFlag       bool
	ChromaSampleLocTypeTopField    uint32
	ChromaSampleLocTypeBottomField uint32
	NeutralChromaIndicationFlag    bool
	FieldSeqFlag                   bool
	FrameFieldInfoPresentFlag      bool
	DefaultDisplayWindowFLag       bool
	DefDispWinLeftOffset           uint32
	DefDispWinRightOffset          uint32
	DefDispWinTopOffset            uint32
	DefDispWinBottomOffset         uint32
	VUITimingInfoPresentFlag       bool
	VUINumUnitsInTick              uint32
	VUITimeScale                   uint32
	VUIPocProportionalToTimingFlag bool
	VUINumTicksPocDiffOneMinus1    uint32
	//VUIHrdParametersPresentFlag    bool
	//HrdParameters HrdParameters
}

const EXTENDED_SAR byte = 255

func readVUIParameters(r *bits.AccErrEBSPReader) *VUIParameters {
	p := VUIParameters{}
	p.AspectRatioInfoPresentFlag = r.ReadFlag()
	if p.AspectRatioInfoPresentFlag {
		p.AspectRatioIDC = byte(r.Read(8))
		if p.AspectRatioIDC == EXTENDED_SAR {
			p.SARWidth = uint16(r.Read(16))
			p.SARHeight = uint16(r.Read(16))
		}
	}
	p.OverscanInfoPresentFlag = r.ReadFlag()
	if p.OverscanInfoPresentFlag {
		p.OverscanAppropriateFlag = r.ReadFlag()
	}
	p.VideoSignalTypePresentFlag = r.ReadFlag()
	if p.VideoSignalTypePresentFlag {
		p.VideoFormat = byte(r.Read(3))
		p.VideoFullRangeFlag = r.ReadFlag()
		p.ColourDescriptionPresentFlag = r.ReadFlag()
		if p.ColourDescriptionPresentFlag {
			p.ColourPrimaries = byte(r.Read(8))
			p.TransferCharacteristics = byte(r.Read(8))
			p.MatrixCoeffs = byte(r.Read(8))
		}
	}
	p.ChromaLocInfoPresentFlag = r.ReadFlag()
	if p.ChromaLocInfoPresentFlag {
		p.ChromaSampleLocTypeTopField = uint32(r.ReadExpGolomb())
		p.ChromaSampleLocTypeBottomField = uint32(r.ReadExpGolomb())
	}
	p.NeutralChromaIndicationFlag = r.ReadFlag()
	p.FieldSeqFlag = r.ReadFlag()
	p.FrameFieldInfoPresentFlag = r.ReadFlag()
	p.DefaultDisplayWindowFLag = r.ReadFlag()
	if p.DefaultDisplayWindowFLag {
		p.DefDispWinLeftOffset = uint32(r.ReadExpGolomb())
		p.DefDispWinRightOffset = uint32(r.ReadExpGolomb())
		p.DefDispWinTopOffset = uint32(r.ReadExpGolomb())
		p.DefDispWinBottomOffset = uint32(r.ReadExpGolomb())
	}
	p.VUITimingInfoPresentFlag = r.ReadFlag()
	p.VUINumUnitsInTick = uint32(r.Read(32))
	p.VUITimeScale = uint32(r.Read(32))
	p.VUIPocProportionalToTimingFlag = r.ReadFlag()
	if p.VUIPocProportionalToTimingFlag {
		p.VUINumTicksPocDiffOneMinus1 = uint32(r.ReadExpGolomb())
	}
	return &p
}
