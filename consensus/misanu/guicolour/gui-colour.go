package guicolour

import (
	"fmt"
)

type Colour byte

const do_print bool = true

func BlackPrintf(bold bool, format string, args ...interface{}) {
	if !do_print {
		return
	}
	if bold {
		fmt.Printf("\033[1;30m"+format+"\033[0m", args...)
	} else {
		fmt.Printf("\033[0;30m"+format+"\033[0m", args...)
	}

}

func RedPrintf(bold bool, format string, args ...interface{}) {
	if !do_print {
		return
	}
	if bold {
		fmt.Printf("\033[1;31m"+format+"\033[0m", args...)
	} else {
		fmt.Printf("\033[0;31m"+format+"\033[0m", args...)
	}

}

func GreenPrintf(bold bool, format string, args ...interface{}) {
	if !do_print {
		return
	}
	if bold {
		fmt.Printf("\033[1;32m"+format+"\033[0m", args...)
	} else {
		fmt.Printf("\033[0;32m"+format+"\033[0m", args...)
	}

}

func YellowPrintf(bold bool, format string, args ...interface{}) {
	if !do_print {
		return
	}
	if bold {
		fmt.Printf("\033[1;33m"+format+"\033[0m", args...)
	} else {
		fmt.Printf("\033[0;33m"+format+"\033[0m", args...)
	}

}

func BluePrintf(bold bool, format string, args ...interface{}) {
	if !do_print {
		return
	}
	if bold {
		fmt.Printf("\033[1;34m"+format+"\033[0m", args...)
	} else {
		fmt.Printf("\033[0;34m"+format+"\033[0m", args...)
	}

}

func MagentaPrintf(bold bool, format string, args ...interface{}) {
	if !do_print {
		return
	}
	if bold {
		fmt.Printf("\033[1;35m"+format+"\033[0m", args...)
	} else {
		fmt.Printf("\033[0;35m"+format+"\033[0m", args...)
	}

}

func CyanPrintf(bold bool, format string, args ...interface{}) {
	if !do_print {
		return
	}
	if bold {
		fmt.Printf("\033[1;36m"+format+"\033[0m", args...)
	} else {
		fmt.Printf("\033[0;36m"+format+"\033[0m", args...)
	}

}

func WhitePrintf(bold bool, format string, args ...interface{}) {
	if !do_print {
		return
	}
	if bold {
		fmt.Printf("\033[1;37m"+format+"\033[0m", args...)
	} else {
		fmt.Printf("\033[0;37m"+format+"\033[0m", args...)
	}

}

func BrightBlackPrintf(bold bool, format string, args ...interface{}) {
	if !do_print {
		return
	}
	if bold {
		fmt.Printf("\033[1;90m"+format+"\033[0m", args...)
	} else {
		fmt.Printf("\033[0;90m"+format+"\033[0m", args...)
	}

}

func BrightRedPrintf(bold bool, format string, args ...interface{}) {
	if !do_print {
		return
	}
	if bold {
		fmt.Printf("\033[1;91m"+format+"\033[0m", args...)
	} else {
		fmt.Printf("\033[0;91m"+format+"\033[0m", args...)
	}

}

func BrightGreenPrintf(bold bool, format string, args ...interface{}) {
	if !do_print {
		return
	}
	if bold {
		fmt.Printf("\033[1;92m"+format+"\033[0m", args...)
	} else {
		fmt.Printf("\033[0;92m"+format+"\033[0m", args...)
	}

}

func BrightYellowPrintf(bold bool, format string, args ...interface{}) {
	if !do_print {
		return
	}
	if bold {
		fmt.Printf("\033[1;93m"+format+"\033[0m", args...)
	} else {
		fmt.Printf("\033[0;93m"+format+"\033[0m", args...)
	}

}

func BrightBluePrintf(bold bool, format string, args ...interface{}) {
	if !do_print {
		return
	}
	if bold {
		fmt.Printf("\033[1;94m"+format+"\033[0m", args...)
	} else {
		fmt.Printf("\033[0;94m"+format+"\033[0m", args...)
	}

}

func BrightMagentaPrintf(bold bool, format string, args ...interface{}) {
	if !do_print {
		return
	}
	if bold {
		fmt.Printf("\033[1;95m"+format+"\033[0m", args...)
	} else {
		fmt.Printf("\033[0;95m"+format+"\033[0m", args...)
	}

}

func BrightCyanPrintf(bold bool, format string, args ...interface{}) {
	if !do_print {
		return
	}
	if bold {
		fmt.Printf("\033[1;96m"+format+"\033[0m", args...)
	} else {
		fmt.Printf("\033[0;96m"+format+"\033[0m", args...)
	}
}

func BrightWhitePrintf(bold bool, format string, args ...interface{}) {
	if !do_print {
		return
	}
	if bold {
		fmt.Printf("\033[1;97m"+format+"\033[0m", args...)
	} else {
		fmt.Printf("\033[0;97m"+format+"\033[0m", args...)
	}

}
