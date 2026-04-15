package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sign_flow_project/internal/dao"
	"sign_flow_project/internal/infra/db"
	"sign_flow_project/internal/model"
	"sign_flow_project/internal/router"
	pdfclient "sign_flow_project/internal/service/pdf_service_client"

	"github.com/gin-gonic/gin"
)

type fillSignResp struct {
	WorkflowID      uint   `json:"workflowId"`
	FieldID         uint   `json:"fieldId"`
	SignerID        uint   `json:"signerId"`
	FieldType       string `json:"fieldType"`
	Status          string `json:"status"`
	Value           string `json:"value"`
	AutoFilledDates int    `json:"autoFilledDates"`
	DocumentID      uint   `json:"documentId"`
	DocumentVersion int    `json:"documentVersion"`
	FilePath        string `json:"filePath"`
}

type versionListResp struct {
	DocumentID uint              `json:"documentId"`
	Items      []versionListItem `json:"items"`
}

type versionListItem struct {
	VersionID      uint   `json:"versionId"`
	VersionNo      int    `json:"versionNo"`
	DisplayVersion string `json:"displayVersion"`
	SignerID       uint   `json:"signerId"`
	SignerName     string `json:"signerName"`
	FileName       string `json:"fileName"`
	FileSize       int64  `json:"fileSize"`
	ActionType     string `json:"actionType"`
	Remark         string `json:"remark"`
	CreatedAt      string `json:"createdAt"`
}

const realSignatureDataURL = `data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAZAAAACgCAYAAAAisjrVAAAQAElEQVR4Aex9fZBcx3Ff97sDgTuRBEXLpiDxSAJCRABkZJeURBGdQCXbf9guya5Uxa6Ky/kj+ohslRzJsSXbIkV9kNS3KClyZEeO4lQqqUrlo+JyIltyYlsSUqGT2JZsR7egSN4Ct0sSAAEcAOHwva/z63lvZnv3dvf27b69/bjZerPTM2+mp6ffvOmZ7pl5CcVf5EDkQORA5EDkwAAciAJkAKbFLJEDkQORA5EDRFGAxFYQOTAuDsRyIwemnANRgEz5A4zkRw5EDkQOjIsDUYCMi/Ox3MiByIHIgSnnwBQLkCnnfCQ/ciByIHJgyjkQBciUP8BIfuRA5EDkwLg4EAXIuDgfy40cmGIORNIjB5QDUYAoF6KLHIgciByIHCjMgShACrMsZogciByIHIgcUA5EAaJc2GoXy4sciByIHJgBDkQBMgMPMVYhciByIHJgHByIAmQcXI9lRg5EDoyLA7HcEjkQBUiJzIyoIgciByIHthMHogDZTk871jVyIHIgcqBEDkQBUiIztwOqWMfIgciByAHPgShAPCeiHzkQORA5EDlQiANRgBRiV0wcORA5EDkwLg5MXrlRgEzeM4kURQ5EDkQOTAUHogCZiscUiYwciByIHJg8DkQBMnnPJFI0Gg5ErJEDkQMlcyAKkJIZGtFFDkQORA5sFw5EAbJdnnSsZ+RA5EDkQMkc6FuAlFxuRBc5EDkQORA5MOUciAJkyh9gJD9yIHIgcmBcHIgCZFycj+VGDvTNgZgwcmAyORAFyGQ+l0hV5EDkQOTAxHMgCpCJf0SRwMiByIHIgcnkwHYQIJPJ+UhV5EDkQOTAlHMgCpApf4CR/MiByIHIgXFxIAqQcXE+lhs5sB04EOs40xyIAmSmH2+sXORA5EDkwOg4EAXI6HgbMUcORA5EDsw0B6IAmejHG4mLHIgciByYXA7MvAB5zWtes3tp36HrS3sPNu7Zd/AbCO+Y3McRKYsciByIHJgeDsy8ADl19tJjJDKPR5I0hP4uwq8DHK/IgciByIGeHIg3N+fAzAsQYnomsIHlD2rVyjdCOAKRA5EDkQORAwNzYPYFSAtr+CstwRggqPQWod47B9cwTgCr83EKq/Nh+IfWNG9kYeRA5MD25cDMC5CEec4/Xgv7uO3o7917/x1Lew+dXdp7oAGV3jp4sBtO24J3CLqrWxjxchvyXlyCbQlOhYs6CJaD6mRp78FBwj6PLO07eH3//lfd6aiIf5EDkQMTyQF0BBNJV2lEpanc45FZ2MdtN/++++67+QY1niOSFxNx+/O/wEQniOmiOoXVKaxOYbgL1PwhSBaHwup8CoXV9RvWdFl6ofmrjeurEE4qkODijEeZE13kwCRxIHtZJ4mismlpsYEYe0jZ5UwBPp15XLiUvgBS3XMXoUsqGJwjOgL70O7VamVPbaVyizqF1SmsTmE4na0cgeQ44Z3LD6EzTJiZrimeHMdV0KgXgk5AgV434/lFjYwuciByoCcHtuwmXswtKysWNEYO7N+/fydmHlWQsAsuv/jXVDA4V60cziM39SBoDkOQ7PHO5YfQGSa8ulLZqXhyHIvM8kcQKM+CGFWxwXPXI9mMJM5GHDfiX+TAmDkw8wLE2j0sPGa+b1nxMHS7fTBXGzuuoNAFOBKS78I/Uj+2/M/gT+KVrq4c/WEIlDshrG4m4jXKfvmMRGcj6+/Ioor/gyeLsAGtZcLI2mwOQFV28IzO1IpjjTkiB7YfB2ZegFi7B+BHtfPYTo8Zhu7H8n0wzWpL8hA65r5nHM2M44Fq1eWXuBmJcJiNiPBAz1Kf/8kzl07DBnQbaqPtXx1AvZxN6HbM1J5XAaNpNTa6yIHIgc4cMC9P5wTTHivET5o67Dy1duntJjyzoHZ+GGGfQwXDSH0KZh4gt+PlZiTC9Gv+LmwmO0+dLT4L0TzI62ZiigtTmgtQlblFAwhfgNML0XLb6XOXA+80MrrIgciBVg7MvACpV5c/zyKq+89qLvSYdq5ZYHb/c0GpBm90hr6eyYO1ArYOn2tSfH2WoOV5OHfJQLMQDqvygOR52Fx2Q1XmFg2ANy/GzKSOeHc1bqSvdED8ixyIHOjIgfIFSMdixhuZcvIZQ8EC1Dpn7j54cI+JmxlQhaObeQg97isVZh4Qpj5uGD8r45DaEHTfhuhZY4hTYTUM2r7yYkb5UZ8QM4lCsxDQuChCb/b5ISwCj/K4lGnuEcp/nPDbwEut4znNm0dHL3IgciDnwLYQIG7kKi1LeHelV+jpe+6576U5H2bGy9UubZ15eTMP7UiNDSHjm8g8hPJjWWC0/+5ZEoVZCFHLjIJ6/XRWBqET1FfCcxs2Kq5Wv/1FIn6aWn+7T55Z/yetUTEUORA5sC0EiD7m2rHKfvhHMOo8D1+vxZTk5xSYJZeajZOo13W4I3mnC3D4q70TDhjtfpsQORoAs4iPe8wiElROPq6bL6nsbN7jtfrKt9/dDDehWnX5Xme0J9aVa+4GMz+sS6FdYHL/ImWRA1vKgW0jQJSrNej/kyR5VGF1wvJeHVErPCvOdqgi6Xu0zmXVTXklKYWZBjryM2XhLoIn4eSyT99vx66dv6b1+RKS93i4g++M9rXq8oKxny1cb+x8a4e0MWqEHECb2wE14tfUKTzCoiLqATiwrQSI8qchbNUWC6fOXD41C+v+8XIt4iU7h04y2AjmJLmodS7LQU31i8wUVEDJHP1Pj3sr99isZmqmlbzshavXd7wlh7t6Vxrz/xg3c9p55Xi18iWEN72E+fd8opTSzyuPldc+Lvqj5cALa5dejxKcy2EE4zUpHNh2AiRXW/iNaUQsL7pBjZWXveyVL6Ep/qlqCeQ3bR9Czx4/3l8niXybXjl/HgoJmeuYjQT1UZvqLCQbHSBfCbgTOhTgLgAT39u8ZfI2IztCMNrbZeCMRLtzXgOM16g5IER/3ZdhYR83Ij9Z2nvg99Xdeed9t4+ojJlAu+0EiD61WrWijeIIMfl1/4tzO+ePq5pD70+jkzRteLpVtQSbz10+PIyvo23dVDe3c+4U8PhjUM7UVpaXoCL7DuLcZWEXMeo/pqahm6n5zZdu5do0bPJ2S5/H5/ajI/lKtixW6NFZmLVmlYn/7RxY2nffa4j4R9UlNzWq+eCJ4m8jB7alAFE2QIgcrq1UXgw474hk8eqNm/4RwlN5JSmHvS7ozH8KlUjhhr6y0bborm0dfZMQPQnefZ8ihj1pWX11FtbwLDnU9zBR8qCp0yJmrdVpHnCYukwIODlk1Fa+/X+Fky8rRSJ8qw6edBClgymNi67JgW0rQHIWoJOV38xhgjrrC0t7D07dmn89ol3m+Hco/81Tsi8Hh/L0hYGaqmk0J3qqXq2ough8I2LmH6D8Z+E8arSeSPjOC1m4W6k2jYW7pW+Lr2d7aOxMZOHq9Zve2ZZsZoPaFrQTVafwVlXU2tYsPOryoer+CQyW/AybiSSeTNCB6dtdgGBEnVwzfEFDod0wFr8wTSqKi5fTD6MO2bMUeq4s24fOPpibRnPiRA3KTnigPLJ2DwvrvdE7NjvKLdytZJvGwt3Sb4zXmQhT8kXyv0Qe2y6zEG0L2omqy2DPhNH6tl1ZeLSlOuxpvVo5iPoGO188mcDxpeUv63RaorZXoJ6PLFFrbw8BSFOjotDRYCrU3M/C0r67WutT2CleMUt2iTrsm2Bje7AwbcHPlmfhLkVDFeFHk5iwcIC7JO8aXasu/7II+UUYO6dZ7dm1kpN0wz5bC28NjSm3nkzws9tlwEB9/ra9AFE+1aqVw3C6gqlVRdHYcXbSZyL5aDBfnorxEiV2RqXVG8jpzms2sw9M5z/QjkhSY7g3cHu6cYdVGEI9GVRxSSLBdjMIbcwScAHv4w7/IIhinonnwGr1219kkeaS8caOtfi8m48tCpAmL6gGQUKU/LaJmviZyFyS3OTpRUM/ns+ofFRhX18O1XMz80dMZtg+lj9vwg6cn5sPI3kLu5uj/rN2DAt3KNedwEt0s7+VCkM14UPFfaEWIb0Alef6NNrOitc8zyFbdyCptXtYOKekbw/teuANiSklnzMF6fO+2PK8zc3tBkYB0vbE0QG/C1GrcFfh9Fq42thxeVIbTCNNX6tEqpOE/1z9YVw2o5Gw6gqW8mcgWDueSouy3+jLsrCPG63PBWwgNi09j2e8QRgWodXlF6O+yzLvPnnm4szuVBfhG1k13f9C1k4cPNI/a/ewcNFCXxhiQ2JdP7wm8pwpkwHv3ioeoKyJvaIA6fBo0GHejRfmveaWazAnJ/NAvSOGTgub6P5BSZvnRQnRk7WV5Y7Cw2Fk04la2N0c8Z8tz8KdirX3mT7ZKUnRuFrzbLVwrMosL2VWoYkZblgqXpRfA6dvfXab7/fpUhDacnNDotDvYkay2CVpx+jasaMvxw07sEQwXlGAdGkDdR11EB2h5mZDglrn/bO8qUgNhKjjw54lc+Q629SH231Jy7CBtGPdPKwvv7QY+DfPM4oUGGgcRqMI+0NsJzWK8saNEzPcoWZu46QfA8KhZ1B43ncnCYfTGBJmvCLjrNX4y44CpMczQIM5XFupwLjOT+XJFrNNRQfdNzDu2nvgTXn82DzbiC1clCDtlK80dpxFvtwgv/l5UdbuYWHgGNmldOpx8mwN/K3qlY1lS8E9Ixsx9BezhbaB/ggqN5VtXxYut5RWbLYcC7em2jy0YQYl9Cm0Jbzbm+e1KawaDfAnJlW1bWkeJRwFSB/crVWXD2B06Q3Gqs4iEpkX4t9d2nfghbvvHt/HqezadMB391GdjklUPYeKhWm9CH2uY0ITae0eFjZJSgdV72yFBwp4TjsH+D0uLmAv6YGmwy1pFV5bZhvoQMrIo9BhBj5aeJQF23IsPEiZaeuH5ZJTZy81V9P1iRDv/EydjdZntbsmiwKkK2tabrhNRczyR4hVY5qecpupdoRfIgkdx0jkClwDbst2sjuVU8I/C5rcNUf8LQcU+MMobLHjqqtMhdcbU0n66d6F9LrLa7VqRXXTvRIRjZBOFV4ttoFZnoWMkI/U7VdimfqsMPDzS3IB9v8tGU+ewwHVdsvZaP7mNvSjAOn/oeffiKi8HJ3WLY0djTuQ9RgcCdEO+PqxogT+7lNnLn4K/sivq6k7xtypnNCJrQyyA11H9JhO9bXqqr1Cko7HBhLoYAmfnw1xYwDaRrYLJ2d4NdYY2FtqkcxzH/cIecCPhOH9P4y8H/B40AF8eisHjqHcCQC0w5sAMqaPhOe+853TaEivyGcl/wc10BUa8HBx8vNoUHLX3oM6K1GnMxPnEHehjO+x68wBDbcpqOb4X6DkQpfiEGOMhiB8qtZr1VUbdmv3sHBbslKCSqvOlFDnTweE1raBSKTpvNbfprMw8gx15ZmzUan8VR4k5uSzeP5bNhP15Y7atzYIC4+qXDzPRTzvwmqmXvSsbtwYeAnlFLaFtKkuGWXufuH85ffA31ZXFCDDPW4/BPgPNwAAEABJREFUK3ltrVpR+8PXiKR5rDqRzkrUKZ+dQyd9S3qFnh1WkGQzh+Y5VakkLytalRNn19/N3MRBbWddbYbP2j0svFm+Qe5n9ZXmTAlIhOfsx8Go+1p/Drp7IgtTaT+h5LcMMgY8c/sErA3CwqjrSK7smTfbp7TamwYuM23dGDiQLSQbNNAREbriCZGGvO+ee+57qQ9vB187te1Qzy2pI4TIGxo70pcS07MocB29yDdzdwL+CQiX84jXi4XolvQyPYuR6vm79x38cY0c2Amt1Ve+/e4i+V++78ArE+KwJJGJzxXFIZL6hQXQJzfhInT0mzYVcao6Tc9EelzLaju9aZrqScGahCyM59HcP2B16i5lOX+hQyH5bjkYJxCL5Z2Ft4BUFqnmPB66NF2ir/gMojdjFhIWkJj4niDe98P1YxW0S/FC5KYb1HhXz0wzdjMZX31ms2Sn2lqp3InGdfNqtfLq3O2Bv6dWPXobOr8nUPPs4EZG10Z0ayr05aW9BxqYlVyFO19YxZXQI8DZ9+WEh/BfIAMaP/6ZnlmtLuu3URDo/7Ib5yzcP4b+Ujp6jbCD8P0w+KszvhYElgYLtyQaYQA0HcaI9KO+iJTEf4DLR0W/AAcs/zCttzO8Alg6J223W51YW39X55R9xKb8qE/FTO/S9urDs+5HAbLFT3i1WnkAHc1uJlJBchKzErwbSgQn6Bhvgrs1hYprae9BFShqP1Hn7CdL9xxMEX9OZyxWB21hxdTL7d//qjtRUAVpXOfGJNXaSmU/woUvHvH3QFSQQqBeSIR16eSCI5BJj1Z5zMFtf+Cd3W38VYwqM922yFxIauEQWR4AWsM7BfjhPfsOlfJlyPIoHByTbWcWHhxj55x4bot47ufBv3AeG+DA1865isXWq8ufh7A/7nMB/0ODqp9qxyuPkXA+y+UF4HpS6dd6ePyz6pf6UGaVSaOoVy5IXupVXujIT6ED9IZ4yBdKEN6JstXpc0owX9H43SlmLGkqn8A9d0GV9HoHbPKnDfrKjeto6Jy4pCzPrlaPDvzxKdAQbAsWdriH/NOXOb3MT4EHtwRULPVewk5adeRGt82BTiILU+k/15kQq4BW3LvmUwmrfjRiUtwgdNhnbOFBcHXLo2301NlLp/Hcb/VpAK9mfPUx5fhQP2m7CJuEMeNpfhahYBG1Y8v7ocdV1bXLCZpvzW04Ljyrf1lHMqu1m4J6eZUXOvI76tXK3YyZCROfYiK1m7TZUOh5CBF/JAOSZBVEx/lGzEz8jOVqDm9Y7aUNmpluynJh7iPJcJ0bZgMeF+iCYAqhoYFrqktmeZEiQkWvgVoIj6NLGu7m6hhVtui2PX3e14wW1vAIXK26fD/QZt8MSejv79v3mmwmhMipvizvLFxipbSNAl022wRAapvDe0Gj+jH9hkctiXgbpY8q5NeOHb0T7bSJg8m130JIpixxMmX0zjy5OjNZrS7fsVqt7IFrs6FUXtaYb+wBE/4EI5x2Y62fsaiAUPgWVYVhKq0qsCvqSyotBwli1vNZxG8QNMA/tmvfoUN3QQDemGP+VU8E6gqbR2/hEdImPAnnNaUQ1NmXC4Xmb8j6T3v6ot8/B3QwAGFc2DbXfwlEmEk5Va7mwQwkwBoexAnxfwz5Uvow2vJ5nVWFuBkDogAZ5IGOMY/OWL7v9sUfxqh8vkmGrHI2a3EzFqIwktLVXqoC2ylEO5l5rpnHQR0FDRq9Ch1nd4GAyeEDjaV7DtRxL8xwEk7Cy51sxO0K6PaX2zfOA5/adQS+usb1y6InvgY6mejJWhU65m6I2uItHQGWrbOBeHKY6UyAk2Q2ZiC+QiPyXUcrFOxbPJd8YURFBbSJ5OpcxAD+CNr7UB1+vVp5G1D5Z4/mS7dCJXdG2zviZ+6KAmQKH2n7NF8o+dRqNmtxM5ZavtqLM6Gi+njIj5aKXmWib3YTNEipQkfbRoKMOYwXjVmPDWnOcBqNcBItRnKfwMunwkYdhAEETraRsgHhEFye5ipmR3XgVj03SEGJ2eXKVBCRSuMTq9XKAQ3360CH6rVd8ibMIY5GbAOh/JemFFaJgY5HXeeY34teZw7kK6EW/N2GMFRCPjQaX20raIfe9kiAb83fr4ELxIDnJcj8PJy/dqG9P612PR8xK76+sLNSl21ZD53m16H7b688Ot4HEuHvF6G9uMdwhFHxdQBPoIHvwv1X16rZsmLOBI2bvTAES+5OwFen8eqfAI4vIa4peBhCBZH5FWY7Wdjdy4UPaTtzDi+oxjkhpOmA7zwxXVQHGOWI2n9UcCiND2iaQs7q5j3sfUVkYQ2PyAm5lWMe+85hOyWPaFZ97VzRXsO+JPTk3fYl9c0CCO3OJxO0YahnNpZzIRqzIOQtvC8k5AeAd0w39kIwif8w3SJUZAMb6YFyIi99qSeSsEhUfxyQHjr/GyRvR8fs9bprqyuVmyA4WjplDa+a2ctq296VPKz2mD14Kd6q4ZoXPCKqbsoIFToLAaDC5pso8yJgCAPSsPrWaRxcEBS31VYqt6gDbpRzVO0/LTRmBUzXfx1CXUTCMlF0Hv45TFdFtoBaVe80ONVz5Xyn/VTtWCWoRwcl4eSZ9R9EXl2h+PocRrDzBYH/sLlTyqnKeF/uTpIkCEUokC+aMmYCjAJkCh9j0O2Ddgsj6C6MnnYv7Tt0nVk+6CL0j+kR9cpy6OwfYKOjTub4o4jLVGgQCIAhDNxGSvWtc2l01RnSlC4oLD8CPAYbiPKZhcPmNx1d6yhb46fNoT0tQg15joSa55CVVIlXvOL+Jah36kCnM1N4UKwSl7IQYn5uLghtC7tC2v7qEPjcOiB6HPVWVVRbymJBqC+D+hTwx4FzpuxhUYAUaw8TkRoN0TbKACtxaKCLJ89cOkkixsiOlzLfI6FpynK96CirjKJ4OtEklITlvxYuirtoetWvYyb2dJ5vUZjekcNT5eXqN+34UJ2MdCmpPV2nhg5sXD+EGds6sB/Rzhz+0Jc9n83C3RCnnHzG3kO9s5V0NrIgLK2qTN2bdFLf0YJoJja5e3ATS10krDMHrB7fwkh96uylX4KtI4zm8MbrsSmlvZQoonnZsi3cTLH1kKUjh1Np6HdcHC3pjcYfOmCL/oToq76olNLbPTxNfkrNI1mERNUwpbUnSWH/ypnBzP8aap/DeXBoTwqe1eYEl1BTLZumavcbio4cp90jNVP2sChAhmoek5XZqUiE3uepgvA4ATXR7jJfSo97WnzlyRzN/TtP79x8Eo478XEj9XMh5sqwsIuY/D/lXyLc1OOn/LGy2lM+En9z4ELJ/IH9YdnjtrCP6+Qn0lxGTNlnGc7ee++9zdMQOmXaJA72nP2ZemyThFN4O5lCmrc9yUG3D0542BkiKT0GA7bX+56F8NBNh0g1msuXrdgtrOGxOTG7f5ledIPk54kl2xHM9BQ6v8e2kjbLFwtvJQ3DlCWZ2s0ZtzEgeVrVcsPg83lVeGC2fBrhsGy3LLUYcJLiF6HfVVgdZoJ9DRzyj7J9De+RP/HhxZeuzZ1QfIpnUGfthQQhNSy+QekoO18UIGVzdAvwtev5dZSo50ah0QfVlRB/cNSktNMx6vL6wS9EYf8FpfQgEwX7Bwn9QT84ykzTxqOp2w+SGrUbeBvUccPwSDvPduEBm91zTt0zDGKTF/aLtyM4kHDCIOMNmDH8K+TPL1nM8eXh4l6jdU/LgtZf+VAc02TliAJksp5Hf9SYqf7cXFK7QWnVj7LRYZ4HktJ01MDV/TJ0QHhZPW/3PIPf6StnvVp5G0aefifwLtCl6/GzvJbeLGbk/9JqRJ0+/bflmYWH4Fz7hkGgWqsdO6qbVAGWf0EYFP6WyGr16Nsx4FgL1Aw5a8i/XdPER7Rw8szFtwb8UwpEATKlD86T3Wikr2OmXT6cEr8fI6jSDJEe7zT5nBg9Non/2M9YqlBvXx46Fir6L1RHxUt7D60t7T3oTg9AJ9pcuiv0CcR/HWl29I+xNaXOlhNjU4GwP472OtLFBb32SrVS1xoS5g+YmJYO/+UvP/A94MWfqHvZK1/Z13LfrJ7ylx4nc/ItD0+rn0wr4duZ7jZd+k82ecEr2mE1w6OFLB0WHm2pm2O3tOAlDedqQU3ShA0a7RDv2nfw6+gM9BgW7TgFsOTHrmic6L4apNttsvUNoiNqqkOYMntM37lHk1BtZqjfhrPIoFq5SCT66WDtG9RhUhtomAd0GGleB3+gi+f4nciY2VSEnqkfq7QsQ8e9Ui7bBixcBLm+S0zy/3ye+bn58AVOnmcdpL0W9147dz05od/ZAbzpVase/X4kekQdBMo34E/H1YVKbSBdbsXoSeVAmz41kClEn6Ut/LXp90fSEQxSHUuXSNq0iXQ5B0s7RIyEtUNQG1J4J8BPDatDnyrzSOdH5k7AQMiosDkHweI6xG60Ak+TBthldBTeLe2o40Gr22SaXqHnQNetKM8KCK17CAO4hvvmYt3w98gwHd/1tKEdqMMpxGGVlIso8c+2AQsXLQI0ft3naaTp74B/7lnf8ZLFrwpTbmjnuas3rn3Yp9vMB/8eVrdZumm4rw1mGuiMNBoO5PrUVcrPkHI+0dbYPQwdKLdp9yhJP27RDwpDaISRIkm6KY14mXUk+Ag6zG/CnUC9LqoD/E0i0U7Tk4Iosu+Mwrs3M7DW2+wyKcmWnomknZ5XS506e2mNRHQm4eqECp3XuqoDHI6cwc0j6CDDkfpqR6hVl5dq1crDuDfQpYITZfxQyCzyvwNcNmDbo4ULlgMBol/D9Ll2qv1G+XnyzKXTLBT4KJygrfhk28fXF2D71HaGaooX+W49Pyq4akVH0DNUw8GrYtf8MyUnqI8f+PnwanYO2B7PUw3Xqkd1FdcRdHzNzlUFN5E/JK8P7ETWLiOJnKct/GUCTrxaClXJCgdQRR3bziJz36HZA34cljRtZCmJIPQ+6+FBfeBQwelG8MDxVFlLgoFrZFcdNizMTsOZZgnsNy+sXX4PM4UVXsT8jKYbGRETjDgKkAl+OJNOWotuWegjOjKbBJqZ+QcCHQkPvbpHO1N0tOE8LxUwScIP+TJa+OAj23ybxsJtyUoNejsHDOGPe8RM9F04FapHUKd9Pr6jn3D4RCuJU191TNZvJM/RRZ8W/Bv6mBCPq5NveWzhTmk3i8vtNOHTtyJilsjzSm1lef9mOGb1fhQgs/pkt6BeTIl9cXZlI10a+8/qvDF6HPpU1/YKqaBEGY/6eMCb2n9sGgt7HGX7qi7SvUFCpHaOgD4lfhCCw80wQmQHQPOz8L8NtxL+sQAPAAzCswGKCVksjy0cEhQEpMsBj7KJ3RH17utI+YLkTEzyaRAgE8OscROCxriY6bIPrSk8bnrS9EbLTncp6YC9oevVovMWvydkaLQeQS4od/qwtO718NGtvqXJwq2pSgnpzKNB6TGzN+gqZWq3vu1kubrJLQ8XoTN12HGGIW4gng1ToOWxhQvi1BSwgOIAABAASURBVPdM3zkm2aDCw0zueB0qrl4oX1i75I6TR5rX5zDA2bmiAJmCZ+kbMQygUAEIdNlyG+DTGj8p5GedzHIpx3CXWidJxeOTlINO38cN66txebNORMvAKNh1xg4mCbCGy3Q6c9CZBwRGEHAp8XtqK5VbVBXXb1nWTmPtN/3mt+lcO5Xm3px+eWZxbDWsQliXOeM9y9+5lsUTjhym5N/QJj80vnCEioU3yTY1t6MAKelR3XnnnQt37j34X5f2HvzT773vvptLQuvO9NEVH5StzcegJ2BeOH3u8liOB9cOAfU8J8Q/4alJEvlfHh63b3Xeydy8nrfkSGJK7YoaFzfIn8XP5psovXDB+BreNcCP6ncweqUf9F6D03eamcdAJzHj+e4moU85Gob8A65FdML6DILROW07Nn3IIjpmt88IdXlM6eiYMI/0AgNC4yradqPLMudLSP48XHYxbbqB0NJh4QzB9P+HRj39VRlvDebmbjmI3v2NoOI1O9cb/xR+KZdO/dmu+IAkCYiFextBQ8JyAaUJGFs21YkwZkaInYArTSXYJNI01c1eGVXMb8qAwf+1I0pTKWT/0NKyFUfilwQn1yT9kMaX6ZQ2EXqvxwn4D4vMOnw+dPh64GToG6SHahJl9tTx520lCA+UIXfcvvAl+CO9GhvOnlq/CMHghAN83b9jnXiBIUQ3gbBQdyY6T5n6bxW8fFGa0u9Q/kvTtHlMTh7X7qUtbVFCu2xPN63hwKhprcCk0H38eOVbaGzu40Gc8Pv37XtNSwdbDp2sZ+l8weNKBc3ZB7bQTzupYJj+yxaS0Lsoo/MWpppPnIo0VxX5yIJ+3iEG9ZD0Y//wZTCH1VAYB/yVjy7DR0euHxI7jcHGDo9PmP/Yw0X8lJt8EpGeev5cr+/0/DncUlQH4cMQUOvoxHUzpu/EFVbXLezju/maV124z5L+QgshBM40hYP2e9aFpHiHj8PpKrV1+E+sVivZMufsu+k0T/RnJvFPoh69j3s3bZEsHJBMN6BMnO4aTA71UDXTf3bkCM3fkPWfJnKh8v5YHmlphGNokKpjhwomLGEtr3KjwcRCP+IxQ4UwdHsHjjmPj0WO1Tcxovq0o/ZVsKGLbI70B9yb0P58Wfi3etEuRD11/Dl/jgCHqtPgtVz6PNT5SIXVtYc1rpfrlt7HQ17TFQgEt1EUftjTozDeKdg5yAuMeyA0dJXazfAfaCLIIAwUdfbUctz7lWvJT2d3t9+/PpTtV+sR1VhITnnUQvybGBUONQtRvSyl9BGPU33bgVlY743aoT6L0LFXUY7bDIb6vgDYXZ1oUfqhU9bzlnR0qM6PEBVuun2HrgP3ULxyROR/bbSoSoLQSQz9LRDQuAiVhKp3SH+czP2++v26FrqENtXL94vXpRMKdjd0isdrA+5NkILf/8BsOywTZuKOH16C6ucwnD5ftyETz+KiOiZyHbnC6rqFfXw3X/Oqa7+P9vldxxv8JXP8fgiEV8OF/TwermULDDoKDGTdcKEub0iEPuxvpESf17bhw9a3z9zCNs00w1GAlPj0RJLrBp1+/zh0Nia+L1BHgullfgovRlixo+qAxo30Lo/Awj5uVL6+IFA9qDE00MNC3+PLSxvpxzCd14MHVUg44QC9cl2IQgeDtNre1AE0l7hzpgbmlcHkQHTynXTNv+FuDvGno3xkzwQSgAZR21lRiOxxNdr18muX9JsVPXL0d0vbCgmFL1Ei13+HG+hK+/z+h7YHPO/z0pAP+IKgIvybHu7ko+M9rJ2267DRaSusbrOwpunl2vP7tJTSv/d0NNLGvR4uw28QnzN4un7fw7ZFC5u8Uw1ufJmnujrjJb4OdYaqNTwVScLHPFzUv0HydrOa5jzyuzX8GEmFr6xZGPdHdmlnkQuPpopES2Nuth9mVe3sRHQzjsjBTHSVYIiE3xxxEq0jbfMqUx3XAVfpS3iF1vIzyZp12ARy6ZHPJ4MJq+HhYfysrZAX7PolyrcNik8kDeeIWbgdXy5MdXCAx+ruXq5XK0MvUnCYSvpj4nAECVPyM/fcc99LqaRfHe86hPYzBt1CzhMTBdC2RQvj1ixc7gWfhYpMSh2E+cuelkFHHNphE8mveDzC9CEdvWkYI/qeOmdNU6ZTWjoKj46FcJ2hllBhoY6h0kNYDZG7dJToR4ZzafLXkB238J9fIpyfbJpHlOwliQx98mtqFg8A/uQgJELkBn25PbNrEFw+DyfStC8whRVi/n4R39Jk4V448JyPo306tSbaS89VWb3wlH0vW/lGT+V4F/HM9CyuPDi8VztW2Y8Bo6p0h0c2pRiiACn5wbXNOt6MF8q9WP0Wo+l13wczhdG+7VytHtXC/eLvN93dBw/ugf3iAoSHzhQCLe35IQXOw+mqlSO16vKSCgkVFupWq0fvQLjFEIn67YYdRVdD5XyRMLtqxz1ouBNfUuGDg+IDzYvgxflEONijAA/07rA5p8vCg9Lm8ono7M+BZOEsptC/pcnCFskr7r9/CaPvsE9EmD/n7586s66n7bpVWTnsb43Fx/sYztzCHPliUSLw7HsKREwhwyIDCCg/C2wWw/SiEBADh8jpBgZ6Caa7yqOlflg9t06D2QgPaltNY2c1Fi6zVi/fd+CV6RV5BrOdjkZRCIyrKM+vWrkNQkJXrfR1GvDJsxe14wntTigp/QuKG/ky3Ie29JmAF6quQbUxNyRazUe3Llzkz9Jm4SI4NqZlY/Ox8MaUm8VYmizs86ka6Np6Q9VczWdoZ492r42FPYIt9m0dLNwvGfnSZCcQc7glqx1IAH5I+eMTODilB30Ybaj5XRgfOeV+aARTXo+JIb9dzz0cYbzxpE+rR7XwcAW53Bht+ZH2k0S8YdYBwXadiTKVVLXS96oVwi9fkXWBKXkL5T8R6rnHIE9W3GvjC8oJI+TiyDbkOFfP9wRsuNNPhKXNwv3k7ZbG4rFwt/S94m1+C+d5MMpWNVA+0hYdSDjbXH6byOaxMI3pZ2mwcBdy9B1YMp/0Rdv5A58UAiCoj31cPpAIajLMrquKQ+9bXgFPkfPENPtUuChARvGYEnpkELTa8CSl5mokll8fBM8gebRsqKtO4yUJI22PB0JDO4rV1ZXKTZhttKikfJpeftuMBujcKP7J+rGKGTn3wlDs3lyShJVSmrMM+4ficY4pLN904W32J3NibFX8aG1GvkOTD3DO4x2Amkv0VAXtG9W59qqPWexMSyNyJ8T2DLhwKnWZ54nlRU2cpwyaOKJmiqA+1/trB97L9oH7qov9OnTPnwj8EXp8ae/BRuYODXRCL/AuIv85vDidbR1QoUFo7EJHMdD0+6799x3C1L5C+YxGiC7ijXwCo/gDNKIfVBU/aFGnw9k/doPnqnazKAeGrX3GwgMj1IxSng3E0mRhLcad35XShxRWh2e5sU2USIuWMayb3zEXlprjOX4abV3fF7fM3MLplU6f+OVLaKvZysEeX/ysV5c/D2N6WPFFud3D8s/Cw9ZpkvJHATKCp9E2Uum8vK+t3BNn198NFZFVGz2nDdMnQwf/OsBqZ5iHby99hnCiJ/TqeT/6gsB1FigQGCqIvoaX52sKn1m7rKu9dJOXxYl3gHRlz5HagBvSFJnOPKiR6qc+QR9ihJ6vVyu3QCAVnsUgd9+XCIVvgKADuFDHC9535raEp9eufBRRGf0ApMsoFLf6uiDcwqzLwn1l7pqIA04iC1Phn6XJw2gnbpBxLW1oJ5nzQsDXSoflwrZ8CxcmZagM+Yziwo1rjfcZRAw4px8QkYVdBLecfbX8IrRVt/EQAyh991yaTn8pc9h7wyk9tPfe+18F/oUVcYDNM+qEYTrjNjBwOqsxWVRrh4URSV/L+/zLmRAHtRcTn0ODbfmSHsLfQC0fYSLtkDH4QwgXwlfg+QtB91IkUBI5gaKrh/Rl8glOrq3rCbrOKHjy9PqbrqfU/HofuV8D//q1ut0os+dLg3QbLl8fCKgGZh5HQahXJ52rHatsevjcBoQDRTS/AYLyBzoPyhebSpp6mEQgAJetuiLc6huwengL942gQ0KLx8Idkm4aZfMD1ueJwYtuINVBhrYvh0IoecgB7X/IE6IsHCJHA2gb17aOdpfCST6j0EUggWYA2dcYmS4SHMInnM90DVSFRSG1lYoefb9xdoVE3a56FcJUSM+qI7S5nRBcf4q0ui8KHt7GImemuRzT8YeOZjoInTYqJeG+Ohpd4YO66cvpngUa9ZOr1eUwgsa9cKFDf3i1Wnk1Gv0v+Uhh0hGWOyICeXW6rTMHf5vRmG/Vl0lfKnUs/J/8zSThH8Os53/4sEjyKyhjHq6w4PA42uoDktydp4GzY53c3bL/JEW1c6QWzqMKebYTTPiThfLOQOJT5y7pydJ2Zqztq9VwPuZ66kwXbVxXDar9zre5nCoJAyw0ivnvvX3xFU5AQEjgXdqjMGx7O9E+Cy0KyZG3eLVjldtZyG8u3OFvivQ+kNKnm0bfdVothMdA+RzoZQeRpv4aLf84GnUx2wBwf9/tiz+KfG6qDV+FkRcouseia33wQr2ZzZfWEk4f1JFc1wxdbugoFcLpHFwDw69P+2TAryO+J/By6sZBHz1yn+fmwplkFh6kYKu7tvAguPYdOnQX+PMpn3euzdjv4wv7pg1hltTcE1IYEZGtYzLHLyNj8wDtJ/Ase89MS6SlF/naTv2MAzPdJ5HWCTkmuoAB1kWE3YwCM6VwvD3iRv4NHUnon6Oclos3OZCyJfGUBaIAGdEDa9OV97CDcNCNosP9b/2QsxluvOTuzCEIE11NEo4z74Bbn786dwvl62ylroJAlzKqYHA3Nvk7sbb+LiRRwaW4GHB+8YOgYaT2jrygFg/65lf5iLQhAfZxRXzgCs/HwkVwKB/B03PXL4uqNZVHLnvjRtr8VomLGfSPA400pA2EKdlP+S+9Ib8M0NGLtvFdjLBbPmGMexsuSemVPhJ5AuzjyvC7zjhY6mhvu3VWgXfAzSjqav+C6tGXO+gz9Pk385lYZ0EmmdTzpb4mbnZA1zhmpzqTUxNtuJvZQbRjEaE3B6o5TH9DVCegH9yaT180+N8H56/L7I2EuQ4YYVV7qfOzFW0TcHKbrgpTGn3mTn62yoqCsRCdhq6yOoG0Y1Fz6OYt1OnVKN9dgJ1e2gUK/uV1L/x8bDGKQ/mIOC9gAeZXQkdyyHuD+abdaHvSMgdDRJSmNzoJibV6tdLWMXYugRNWQ7u7yVLu+VNaLxXEG2YcROtE6KhXji65gtv+hORjPkqkedaXjyvVT+VvteLj32wNz1YIHcVsVWiSaiOb2EHUXsDc+ciSzeoBS3fPIxRUeOBF+wvg8Ya8NYzKFjFCyz6Qs1LR1VBe7aX+behsn0D6S3DuUtpgQNWVXfmyx9aVXdpZSyP9cyJ27Qid17PoaBxelDWwHYWG+N0geTuye8M9CaVfQXigS58PMjrVCHxoh9jsgdCYzR1a3wotAAAQAElEQVT490tsnjFynIUr9RKzMkzLyukuVEauErogxLrIwue9qm0Cz/J2H7GZj7SPQYWUbaxjeVH+PDbL1tf9vF4qiLP0fsZRrdxcq3YWHprQnullYb1XxEGAtaxg7JSX57h90cZDyJcf29Mpx3THuRd/uqswudS36JOZg25aG5SqiNC7BXsBUbHjNiAcwrMD/Khbo09EGe6D5xB3FEG3Yxh2jipe7E07AQiXB5DuRURsR+3oQyj/yW2nzqxfwCjQCZQGp/p9aCeghGitfqxyZ55wnJ6lnebn5vXYjYHosc8Ps8lNPx6Vd8It3z9BwR+EcxcYeSJJWJcFu7DF7yIG/KurmgbtJ2QX+hTaQbOjDTdaAaTRkwfUdmVXLYVEoPcr2iZCRP9AODq/5aDH/vO3pFQ6O7wv52pdZhwtmRHgks4fe2Htklu9CJSvx8DgMuhq4THC+r2YMBtHOr12nT53+R0KzKILndAsVG7S6mD1rRbORlKi9gm8oxnVQvRZKvDL9KpSz7Mk1yR1G7xOnF1/N+K0YWe4mZ5erR7dh7i+r1p1WYWNN8Srekt3omf5jSDMIvw/f8BD4/XTv7TlN9L0jTZcBLbPTJjvgeBs5M4J0BzWOBdOr1Cv75+cQWe8x+K0cBG6OqVtaz+bfotGhR06wTPIt7sNny5pdVG496PaKbrAGP+6vC8P90uS5bOF+83v06WtG1LnwL/H/D31MzrJDag07F0jLfdbJB7vJPhRgIzyKbCxaeSwvpAwNIaGh15ev5o2mL2A+fEm+fJXqlJKJHlfM45WYVAcaAVUrVrxhnhVb+lMBgKFTwI3+hX8N68LAAejHxnLvjDCb91omfN9kHJk49p9fV/UeXQKq7NhhRt4rtfgIHzlFHxdifYSvUGWHgu7m4P/tc9CROjN2tbaMWocBN95CDs9EVmfq0+iND+BOqvh3MftzDtFH95yX+kd+n2xfLZwwdpkPCZdCJHnlO7f+xEKqkou+VskNEE/2/gniKwZJEWyT5jqC8lGJ54SP6id9dA1FvoIVErP+Y9QoaPST7gW2gzViwalEfg/hzToD/HvL6YP6j0fHLcvRBsOvBuUprzDOAJeXlSHikMg0AmF1W0MB2Exj9nGTjgI341H2g9Kz2b5UPcwi9U2durselCdaEcMwaFH1lwEHjWIg3xAJLp44gieodL8gNYZ6rpmJ5m3W03Z243m7sjel8HJ/b1mVg6r35S/BF6Fewk9CiHu94Qs3qDGu8K9GQKiABnhwxRj3EQxC2e/e/XdbaOpqr6wuDfQ1aZD19Eke0Qou6+NjD79Zn42u+GH2tOhnMKG5XYcpYalua9G8bbxSKMKOXSshzGLu0VdJhAqexRWtzG8ubCw9Fi4EFFdEudtSe1SeQr+BIRGA06gbllHpKqrQhtB+Pla9ehtWkfA4Uo5+UwIEC0g72nXQZrInqB9BhbumanLzdb8g70vQrDr5fjZwHlUQe+ESa8zchc81bbhUt8LFvptdxN/zPxeb6dEcGauKEBG+Cjratw0a9BvXE9153dY1YPZx58NU3xDuN1ofQW9gzbw0lVKmH28BbS2ryZpOa8L9yfg4jAqVGIaN9KR7EVQ3IM4q4O38CC4OuURahrpcZ/hOr3jatNaheDoeLRM3m6fQ15/FRQibJ6BhT26In5LfjP67x+HEDVn4ik9qIOh/nO3pmSLS+h9iksdAW9IKXRCedjNThnSzQDQqXHNQLUmqAotx180z2gChVfQyH4Kvl4DOfftEaJVIlH99QV0CAs6KoZf+hJa4HwML88TTHxK1TcgWDuglvO6EDf2i4VfsESA3rAvwcaPDWbyag0CH5swlfNDm9KZJ+xV5NRtTHQC5VxU52AiVVftwvNsdqodiq4dO6rP1q5oW1B1UoekG6PKrGMJuOrVyttE6ExO6K6URL9pkgeLeXi/3oYcmX2DaRcGVu+EU5WfagBIy6kdq+xBmuxqs1NmkbPzHwXIFj5LYdrhi0tFPuzhYXztCGrVo6q/3j0Mnn7y4uV5YLW6fIeqb7TcfvJsdRrwWE8tzouV09koMA9uEw/Pxi6AaFG56b1+2YC0t7NIMBSXrXLrl44y0nFCYeGK/U7HILhF+EMm3/2AnfCAT8Qt9wizzHAvJQkwzcgvCpARP8i5pPlxIxb6MV8cXsZNea96Z13/rjrs3Kk+W51bNoo4hdWVFT6nZXoap9FnpmuebuZEN0b64ET4eO5znhAL+7hJ84X5y54mdIafRJvbvI1Yu4WFPaIivs1v4SI4kNby2sK4RVTwjynVs7d8rubGS6ir66q29nfgJ8LhPQf86KzZQULlUNd4jYADjTS15x1Bi0CEvycxugsjImr76Rp9PSgOxsuLUE/pfhGfQp+XulGFd2uZfXUSnoIJ88V8C8TCk0ImOuFgH7DwpNDXTodsXMq8e3NVFoc6ElmYBvjZ/BYuhsryGvCjQw2UmN9kSmcPQ1aETaI+LpsBm/1aaePpocr2iCfEt53RhJA0c2QcsTVikqOr1coBG6ewNqp8ttFIr2z8OhpaqR4r4fTakEAX1SGunDCRGlWVDHVAS310Epp0Ep14XTeIszCCk3CVoNPfymrkI2ptw802ItmS9K50lFnHknC1CcKh9re04crYIPRMzqssbP6Fkk+Y4E2bC2CTesLBKEBG/ICkdSkvpULL7UWq8MgO3BOdbdhnokdSn0B6/cDTLgiePeqcDcKcZTV0uFpR3ewRIbmIstzVv77WJY9/M8wBzJYPC/F7TBUXMFMttrTXZC4EilmWbeFCSIi0c4c9R43dBXNuTK64EBsWKigMw3k4xRjhlsulF1nxkZKmDQ9Pu287q2mvy0TSnzUeaq624eRrnlAVHDrrwMu4Dt19WN6LKQD0+FLHi6tHUu+BX/qqKk+D97UMTvljPgx97UfuvfdVe314WnzmJNgYLDwp9Fv9u4Unhb5udGxox9Rjf4jt6C3cDXmveOFwDA8GY0Opnngu+YIvalje6/uigzl1Cnu83XxOWA/5dLc5SR7Td98FpvwvmXL6p4L82rGKjk5UDeD2Z2jjyQUHRvyis45mPZifQaPcWetxumgzcbmQ6msxCznlsV6+fv1JpdWHp8EXScPyVAtPCu1pKsE+YOFJoa8XHa4di7TsD8m/BdOWjUMdaUgbiCQJBlPkfhhkDaV6svy2sEM+4r/UnqMldHNnvhUnYtw5ogDZoiegoxR12iEbdRX74gFcBXyktrKswgZgdukmJTWoq1PjehY7yv8knCYqQjumTl8raXO2Z+FRsqwI7pJ0+kWKLDOt2x8idI7yH2aqD2kbzYPuNGi0m6G+oeJxqZ/tdeKwH0XSTewvmqmbGyPv69Xlz4MvYU9SO9+6kTzp8VGAbPET0vOJMJKy6ip3GCFmHbq5a4OqqiHpW4ToVnUwrj9tX9ZRkK4NvSxd8Sjo2xRnkqjNKEtm4SxmJP8YFGz6nYiRFDwmpJiJvBhFPwWn12KD0+f8yj0dcNj2LW02QM1Q1KHtf8DnUdxahg9Pk18/VtGZWeAb7IwDb2iclHpHAVLgSQzbUSD/Il6oMMJH0c9DcOzWmQngjpeqlTA7eTq/uShM78jhrp6WoyoydQp3TdjlhtgPYQl96hX339/xS29dso81mol1B7WjwcIuov2vpPCpM+s/BFTuWxE5jGDny+reLdw59eTGCrHuePcEMoDdp85eOo2RdRgcIW6ws6uQ0V46qCHiYIQmoceX9h26jrZdaPOs5beFaQt/ScJf9MUl83wZgvdr6lCXsMnY358GPwqQAk/phbXmB2VyuK/caByL2pnjBbuIEdROn0k4+Q8e7uUL0Vf9/ZRS/VaHD27wtayTZ9ZPEwlsK3LbC+cuvXNDok0iIOTsAYnJtfXGd0Y989mEpL5vpyK3+MQW9nEj8e2+AAt3KMzq3i3cIelER2WdOh1BW9FDGj2tCymJHSANdHaVR2Z9af9ejsg83qeue6lsXg9bflvY398KH+WGJfyNNHWDDpT7+iL9CdJPzBUFSIFHgUYcjgq38GYosim3oEMnHanlyXkt0+/mwV5eAd2tlsXMYRQoQn+7F+pO93znACGkdhlNMtT5QYpgq1y9WnkDmKx7Zq4qvCXlFng+aAFNG43NtyWElluIzpzrx5ZvBtZgowAM9uNfrxLrV4cNASh16ax+PwcgWmeXb564m53+LD0W7pR2RHECsgNqoR/xMOJD3+LjpsEfQoBMQ/XKpRGjB90v4ZBipBVgF9HlT2cEavjzt/F26QtwpJZ99c9Hj9LX1V+F8deqlcPE/Gs+47DnB3k8W+FDLah7Zvp6PltBz6yXgbais2JtZ2rPC9WVEuwfARkAlHM4JX4QoLuYaUEXpOg75iKm4K/eerDjjikguSeJUYD0ZE/rTeh3A78A93WuDaamv6AN3WPSF0BfBB/ux7f6Wgu359UXyQqr9vtFw7YsCxfFs63Sq35+76E1fRad6m35aOFOaacpTts0nNokdJagCxnckvWy61DXmQjzMx6vvltQZa33YxOx/Lawx7VVPpuDHX2Z+t52azM+zST6oUOcROImjSY1aFNzHXxyTRof34xG6OH/jklz1r0AJqIfEDMfXb3hkgLuupkqU19RUF9phmFGgSirr3K1nO3sNvJYbtNn0YknbTwN/O2UtlfcpN6DEHEnAas/KhrdUncjRFw5ahM5c+mjDu7yNym839heCJN9WujWZmiCf1GAFHw4wtw0DAr9vV7GZb3HkrzBFPG4gfsGpfVAu46bqXT0oqOYFqQizw8isDyOfsr1abezn/NYR95h05s+C30mG/hide8W3pAwRvTigBMiRFCb8aWQjunndLFKR75rIstvC+u9LXTaXkQk7AnZwqJLLyoKkIIsrbfqMHsal6/pd5BZss9posFiVFZo1YgnrY5pe8vejA6H2Z04u/5unc77PM5v+ZiViyn0t6HcQrm3V2I8W9XP/7KvtT6LadPPe9qnxVeeC9GvGnphYsTs7+yl9aW9B/UTB/qpA3UKCwl92qQdK8jCvzVWAkoqfHsKkCGZNzfXPDNKWN7bPuLRMBrwuTnmZuMW+u1hik05+YzJv3D63OWwH2T//lfdiQf5iLnvwE5TZXejwB+XeH5QgWKnMqkKXOgiWvXzZy6dsScIJMzhrC4LT2WFJ4Box3PMRCA5Wgz4OWl4LUhdHiQky8Ay3o0M02D/Th1O5DcVOiQg7iYHTNGfZe4UkT1eUhvCdxoKFmDEazmZ9OSZy2/FfTUowsPFVMNoaaDZB3K7y70oImqcdOFG2rjXAfi70rj2MSKY5sj8RJ5zeUzUIOCk6I0HoX0ceXLVSvO8KKZd6RWq64BCBxaRn+U/FbxbanfR903ViCcgJi6qQ4fc+XMHEDhlvBvD1kRaN2NSKvw3hsW51fmjABmA4/n+Dbv2vUWIJInYI9tfqK1U7hqgmI1ZhH/dRzIlP3PPPfe9VGcfeFH+AdmfkUuPUgAABd5JREFU0Fot+6a1jR0MhuotZLRwiIxAOwfQoelu+FU8F7+PRt+z3XqMDTq2MENpgduRzG54ZDUD31WQ7MH7dou61WpljzqF1SmsTtONjIgCiHMhdoSZrqGtnEf4pwpkn4ik2rAngpBpIwKN8HYSanYGerz12iV3ZLMQNTcFMfVcGVKk3m3T3sWU5Oeuy/WPELXOPoT5AxR/Y+UA2sfd6Kx2oWM4EQgRfowTvjWEI7DtOYB2cnh1pbITbUU3Gk8dP6IAGeKR1Y5V9sO4HVZTzCVJpsOUcj6C04m0xJylA/vLr6Qp/cOWdBBqGMnYc4pabhcNWD29hYviKZJeVT1Lew+tqVO4SN5JS5sSQ8DnVDHdJA35YB6ireKnLy/6kQNlcyApG+F2wycJ/7mvc9pIH8k6PDbr+y3sUw7uo9N5pcndsttahK6oUDP3hwbHobPP1sMLRmTSdT/F0BXbIgS5MD/CROfbi2zcSO2zbL8dw5EDE8+BKECGfEQi/McehRDtUIO6CL3Fx1EJdgNdxXPX3oMXYIht3Gikbwu4WwGpH6u0bCJsvT1gyNJv4QHRbcdsTk1RrdwGIfIEnLeLoGlwmL1uR77EOk8/B5Lpr8J4a+BGmFAbGSoWmCnMDKTP84B05rK09+A5uAacW7cOX+Gr6RV5FsJJT5nt8rxkBZ1Ul3uGsgiOlQPQcz8AtwtCRAXJE7lNa6w0xcKLcCCmbedA7HTaOZKHe3To2qnbDr6BoeQr8mwbPCb5bC4INJ93LflxXzBzWUdmXYpon4nCsKtAJOGmCCENN3feIg6d0Yla9WjX8pFk4Et5QEKPeQRQn4U9DD6ubB9l7kaZnyob7yThgxBRQfLAJNEUaYkcGIQD2kENkm+m86ATW0SHfhqV7NShW54prA5Ju156v935xD7ehzv4EBt5LITROWKBQMkj4GFmcgcEUAMqrivwry7B+Hz3voM/jltDXYYHQS3Wtv9lKPzdMoPvKrCULy6J9DmDc4njX+RA5MCWciC8qFta6oQXdvLsZbVhhI5zvORmsw9HA/PLMTqfd3DzD5MQSiBI9ENVEC5yWyr0ZQgTN9vJBYvCksMqaDScCZ17DjybpT2gcVfyNH5G1MKDnYs8zFEQTYp7QS12Fl6pV5dLW1HWq9h4L3IgcqA4B6IA6cCzvNM6gluX0Ttfg4qq187WbyLNCU2jvjrkE3T0DcB6T53bEWvDeXq9980c1jQaVt+6EAe8X4JrOf4A4ewS0vK+BVz+a4L6bL1gURgkkQoZdRrO7jG/DAgQZjjamQsiRG28rq03jmfCxp0zpGo4FTrqpJ/jtDdi3BiTigShlUr6LzemiDGRA5EDk8IB7TQmhZaJogNG6cNwi9BX79RdrOoAd9rZ+mqNb7lfrSS1Y5V5xOs9dS6fDefp9d6rc1jTaFh960Ic6HkrnC79fISI62R/THPo/L8fUqLMZyoQeteI6AqcXgiSxa+wOiJxnxhd6ypgOhxuh7QqhNSpENIFBOcTYlVhaVkEm0vigPgXORA5MBwHRpQ7vqAjYuwo0UKIPFyrLi+hDN1fYA+Ra+/gkaTQdQEI3OwHuY7UIAgh9HbCX0D8E0x8ioncbAsznYsOJupVvrYvdUDnLoXVuYD50zi1NzV3afPw54cZ/BGMHIgcGAEH9MUdAdqIcis4gI79MDp47XhVkLiO33XqTL1Ubi0CQNOrA70qMHYDn5v9KG7EhQvxD6xWl+9YrVZazhpCuoHLVyGEAq7DtV18DrOycs4Pa8Mcg5EDkQPlcSAKkPJ4OTZM6MRVkLiOv72D7yfs0ug30AeswaDlQ0jcgrww/NOqCpOmIFt+8YCkjChbRBs5EDnQiQNRgHTiSozbUg5AiNytwmRYQbalRMfCIgciBygKkNgIIgciByIHIgcG4sBWCJCBCIuZIgciByIHIgcmmwNRgEz284nURQ5EDkQOTCwHogCZ2EcTCYscKIEDEUXkwAg5EAXICJkbUUcORA5EDswyB/4/AAAA//+w66ONAAAABklEQVQDAPGtpwMhV68LAAAAAElFTkSuQmCC`

func TestWorkflowSigningPDFIntegrationFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gdb, err := db.PostgresSetup()
	if err != nil {
		t.Fatalf("init postgres failed: %v", err)
	}
	if err := gdb.AutoMigrate(&model.WorkflowSignerModel{}); err != nil {
		t.Fatalf("migrate workflow signer failed: %v", err)
	}

	engine := gin.New()
	// 使用完整路由，覆盖真实 handler/service/dao 链路
	router.RegisterRoutes(engine)
	users := seedWorkflowTestUsers(t, engine)
	userA := users["A"]
	userB := users["B"]

	mockPDF := newMockPDFServiceServer(t)
	defer mockPDF.Close()
	t.Setenv("PDF_SERVICE_BASE_URL", mockPDF.URL)
	pdfclient.ResetDefaultClientForTest()
	defer pdfclient.ResetDefaultClientForTest()

	title := "pdf-integration-flow-" + uniqueTestSuffix()
	created := createWorkflowDraftViaAPI(
		t,
		engine,
		title,
		userA.Token,
		userA.ID,
		[]uint{userA.ID, userB.ID},
	)

	// 场景 A：draft 后 initial 版本存在
	assertInitialVersionExists(t, created.DocumentID, created.WorkflowID)

	sigFieldID := setupFieldsAndActivateForSignerA(t, engine, created.WorkflowID, userA.ID, userA.Token)
	signatureValue := realSignatureDataURL

	// 场景 B：FillSignField 生成 v2、更新 document/version/date/signature
	fillData := callFillSignFieldAndAssertV2(t, engine, created.WorkflowID, sigFieldID, userA.Token, signatureValue, userA.ID)

	// 场景 C：Submit 仅推进流程，不再增长版本号
	assertSubmitNoVersionBump(t, engine, created.WorkflowID, created.DocumentID, userA.Token)

	// 场景 D：版本列表接口返回 initial + v2
	assertVersionListAPI(t, engine, created.DocumentID, userA.Token, userA.ID)

	// 场景 E：指定版本预览可打开
	assertVersionPreviewAPI(t, engine, fillData.DocumentID, fillData.DocumentVersion)
}

// TestWorkflowSigningPDFRealIntegrationFlow 真实联调入口：
// 1) 需设置 RUN_REAL_PDF_INTEGRATION=1
// 2) 需设置 PDF_SERVICE_BASE_URL 指向真实 sign_flow_pdf_service
func TestWorkflowSigningPDFRealIntegrationFlow(t *testing.T) {
	if strings.TrimSpace(os.Getenv("RUN_REAL_PDF_INTEGRATION")) != "1" {
		t.Skip("skip real pdf integration test: set RUN_REAL_PDF_INTEGRATION=1 to enable")
	}
	realBaseURL := strings.TrimSpace(os.Getenv("PDF_SERVICE_BASE_URL"))
	if realBaseURL == "" {
		t.Skip("skip real pdf integration test: PDF_SERVICE_BASE_URL is empty")
	}

	gin.SetMode(gin.TestMode)

	gdb, err := db.PostgresSetup()
	if err != nil {
		t.Fatalf("init postgres failed: %v", err)
	}
	if err := gdb.AutoMigrate(&model.WorkflowSignerModel{}); err != nil {
		t.Fatalf("migrate workflow signer failed: %v", err)
	}

	engine := gin.New()
	router.RegisterRoutes(engine)
	users := seedWorkflowTestUsers(t, engine)
	userA := users["A"]
	userB := users["B"]

	pdfclient.ResetDefaultClientForTest()
	defer pdfclient.ResetDefaultClientForTest()

	title := "pdf-real-integration-flow-" + uniqueTestSuffix()
	created := createWorkflowDraftViaAPI(
		t,
		engine,
		title,
		userA.Token,
		userA.ID,
		[]uint{userA.ID, userB.ID},
	)

	assertInitialVersionExists(t, created.DocumentID, created.WorkflowID)

	sigFieldID := setupFieldsAndActivateForSignerA(t, engine, created.WorkflowID, userA.ID, userA.Token)
	signatureValue := realSignatureDataURL

	fillData := callFillSignFieldAndAssertV2(t, engine, created.WorkflowID, sigFieldID, userA.Token, signatureValue, userA.ID)
	assertSubmitNoVersionBump(t, engine, created.WorkflowID, created.DocumentID, userA.Token)
	assertVersionListAPI(t, engine, created.DocumentID, userA.Token, userA.ID)
	assertVersionPreviewAPI(t, engine, fillData.DocumentID, fillData.DocumentVersion)
}

func newMockPDFServiceServer(t *testing.T) *httptest.Server {
	t.Helper()
	type applyReq struct {
		RequestID  string `json:"requestId"`
		WorkflowID uint   `json:"workflowId"`
		DocumentID uint   `json:"documentId"`
		SignerID   uint   `json:"signerId"`
		SourceFile string `json:"sourceFile"`
		TargetFile string `json:"targetFile"`
		WriteMode  string `json:"writeMode"`
		Fields     []struct {
			FieldID   uint   `json:"fieldId"`
			FieldType string `json:"fieldType"`
		} `json:"fields"`
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/pdf/apply-fields", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req applyReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":      400,
				"msg":       "invalid json",
				"timestamp": 1770000000000,
				"data":      map[string]any{},
			})
			return
		}
		if err := createFakePDFByFileKey(req.TargetFile); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":      500,
				"msg":       "create fake target file failed: " + err.Error(),
				"timestamp": 1770000000000,
				"data":      map[string]any{},
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":      200,
			"msg":       "Success",
			"timestamp": 1770000000000,
			"data":      map[string]any{},
		})
	})

	return httptest.NewServer(mux)
}

func createFakePDFByFileKey(fileKey string) error {
	fileKey = strings.TrimSpace(fileKey)
	if fileKey == "" {
		return os.ErrInvalid
	}
	abs := filepath.Join("storage", filepath.FromSlash(fileKey))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	// 最小可识别 PDF 头，满足预览流与文件存在检查
	content := []byte("%PDF-1.4\n1 0 obj<</Type/Catalog>>endobj\n%%EOF\n")
	return os.WriteFile(abs, content, 0o644)
}

func assertInitialVersionExists(t *testing.T, documentID uint, workflowID uint) {
	t.Helper()
	doc, err := dao.DocumentDao.SelectByID(documentID)
	if err != nil {
		t.Fatalf("select document failed: %v", err)
	}
	if doc.CurrentVersion != 1 {
		t.Fatalf("expect currentVersion=1 after draft, got %d", doc.CurrentVersion)
	}
	versions, err := dao.DocumentVersionDao.SelectByDocumentID(documentID)
	if err != nil {
		t.Fatalf("select document versions failed: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("expect one initial version, got %d", len(versions))
	}
	v := versions[0]
	if v.WorkflowID != workflowID || v.VersionNo != 1 || v.ActionType != model.DocumentVersionActionInitial {
		t.Fatalf("unexpected initial version: %+v", v)
	}
}

func setupFieldsAndActivateForSignerA(t *testing.T, engine http.Handler, workflowID uint, signerA uint, token string) uint {
	t.Helper()
	fieldsBody := map[string]any{
		"fields": []map[string]any{
			{
				"signerId":   signerA,
				"fieldType":  "signature",
				"pageNumber": 1,
				"x":          10.0,
				"y":          10.0,
				"width":      80.0,
				"height":     30.0,
				"required":   true,
			},
			{
				"signerId":   signerA,
				"fieldType":  "date",
				"pageNumber": 1,
				"x":          100.0,
				"y":          10.0,
				"width":      80.0,
				"height":     30.0,
				"required":   true,
			},
		},
	}
	putRes := performJSONWithAuth(engine, http.MethodPut, "/api/v1/workflows/"+uintToString(workflowID)+"/fields", fieldsBody, token)
	if putRes.Code != http.StatusOK {
		t.Fatalf("save fields status=%d body=%s", putRes.Code, putRes.Body.String())
	}
	actRes := performJSONWithAuth(engine, http.MethodPost, "/api/v1/workflows/"+uintToString(workflowID)+"/activate", map[string]any{}, token)
	if actRes.Code != http.StatusOK {
		t.Fatalf("activate status=%d body=%s", actRes.Code, actRes.Body.String())
	}

	fields, err := dao.DocumentFieldDao.SelectByWorkflowIDAndSignerID(workflowID, signerA)
	if err != nil {
		t.Fatalf("select signer fields failed: %v", err)
	}
	var sigFieldID uint
	for i := range fields {
		if strings.EqualFold(fields[i].FieldType, "signature") {
			sigFieldID = fields[i].ID
		}
	}
	if sigFieldID == 0 {
		t.Fatalf("signature field not found after save fields")
	}
	return sigFieldID
}

func callFillSignFieldAndAssertV2(t *testing.T, engine http.Handler, workflowID uint, fieldID uint, token string, signatureValue string, signerID uint) fillSignResp {
	t.Helper()
	fillRes := performJSONWithAuth(
		engine,
		http.MethodPost,
		"/api/v1/workflows/"+uintToString(workflowID)+"/sign-fields/"+uintToString(fieldID)+"/fill",
		map[string]any{
			"mode":  "draw",
			"value": signatureValue,
		},
		token,
	)
	if fillRes.Code != http.StatusOK {
		t.Fatalf("fill sign field status=%d body=%s", fillRes.Code, fillRes.Body.String())
	}
	var wrap apiResponse
	if err := json.Unmarshal(fillRes.Body.Bytes(), &wrap); err != nil {
		t.Fatalf("unmarshal fill wrapper failed: %v", err)
	}
	if wrap.Code != http.StatusOK {
		t.Fatalf("fill sign field code=%d msg=%s", wrap.Code, wrap.Msg)
	}
	var data fillSignResp
	if err := json.Unmarshal(wrap.Data, &data); err != nil {
		t.Fatalf("unmarshal fill data failed: %v", err)
	}
	if data.DocumentID == 0 || data.DocumentVersion != 2 || strings.TrimSpace(data.FilePath) == "" {
		t.Fatalf("unexpected fill response: %+v", data)
	}
	expectedPathSuffix := "/workflows/" + uintToString(workflowID) + "/" + uintToString(data.DocumentID) + "/versions/v2.pdf"
	if !strings.HasSuffix(strings.ToLower(strings.ReplaceAll(data.FilePath, "\\", "/")), strings.ToLower(expectedPathSuffix)) {
		t.Fatalf("filePath does not match version rule, got=%s want suffix=%s", data.FilePath, expectedPathSuffix)
	}
	if data.SignerID != signerID {
		t.Fatalf("expect fill signerId=%d, got %d", signerID, data.SignerID)
	}

	doc, err := dao.DocumentDao.SelectByID(data.DocumentID)
	if err != nil {
		t.Fatalf("select document failed: %v", err)
	}
	if doc.CurrentVersion != 2 || doc.FilePath != data.FilePath {
		t.Fatalf("document not updated by fill, doc=%+v fill=%+v", doc, data)
	}

	versions, err := dao.DocumentVersionDao.SelectByDocumentID(data.DocumentID)
	if err != nil {
		t.Fatalf("select versions failed: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expect 2 versions after fill, got %d", len(versions))
	}
	var v2 *model.DocumentVersionModel
	for i := range versions {
		if versions[i].VersionNo == 2 {
			v2 = &versions[i]
			break
		}
	}
	if v2 == nil {
		t.Fatalf("version_no=2 not found, versions=%+v", versions)
	}
	if v2.ActionType != model.DocumentVersionActionSignApplied || v2.SignerID != signerID {
		t.Fatalf("unexpected v2 record: %+v", v2)
	}
	if v2.FilePath != data.FilePath {
		t.Fatalf("v2 filePath mismatch, dao=%s resp=%s", v2.FilePath, data.FilePath)
	}

	fields, err := dao.DocumentFieldDao.SelectByWorkflowIDAndSignerID(workflowID, signerID)
	if err != nil {
		t.Fatalf("select signer fields failed: %v", err)
	}
	hasFilledSig := false
	hasFilledDate := false
	for i := range fields {
		f := fields[i]
		if strings.EqualFold(f.FieldType, "signature") && strings.EqualFold(f.Status, string(model.DocumentFieldStatusFilled)) && strings.TrimSpace(f.Value) != "" {
			hasFilledSig = true
		}
		if strings.EqualFold(f.FieldType, "date") && strings.EqualFold(f.Status, string(model.DocumentFieldStatusFilled)) && strings.TrimSpace(f.Value) != "" {
			hasFilledDate = true
		}
	}
	if !hasFilledSig || !hasFilledDate {
		t.Fatalf("signature/date fields not both filled; sig=%v date=%v", hasFilledSig, hasFilledDate)
	}

	return data
}

func assertSubmitNoVersionBump(t *testing.T, engine http.Handler, workflowID uint, documentID uint, token string) {
	t.Helper()
	before, err := dao.DocumentVersionDao.SelectByDocumentID(documentID)
	if err != nil {
		t.Fatalf("select versions before submit failed: %v", err)
	}
	submitRes := performJSONWithAuth(engine, http.MethodPost, "/api/v1/workflows/"+uintToString(workflowID)+"/submit", map[string]any{}, token)
	assertSubmitOK(t, submitRes, "submit after fill")
	submitData := mustParseSubmitData(t, submitRes)
	if submitData.DocumentVersion != 2 {
		t.Fatalf("submit should keep version=2, got %d", submitData.DocumentVersion)
	}
	doc, err := dao.DocumentDao.SelectByID(documentID)
	if err != nil {
		t.Fatalf("select document after submit failed: %v", err)
	}
	if doc.CurrentVersion != 2 {
		t.Fatalf("submit unexpectedly bumped version, got %d", doc.CurrentVersion)
	}
	after, err := dao.DocumentVersionDao.SelectByDocumentID(documentID)
	if err != nil {
		t.Fatalf("select versions after submit failed: %v", err)
	}
	if len(after) != len(before) {
		t.Fatalf("submit should not insert new version, before=%d after=%d", len(before), len(after))
	}
}

func assertVersionListAPI(t *testing.T, engine http.Handler, documentID uint, token string, signerID uint) {
	t.Helper()
	rec := performJSONWithAuth(engine, http.MethodGet, "/api/v1/documents/"+uintToString(documentID)+"/versions", map[string]any{}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("version list status=%d body=%s", rec.Code, rec.Body.String())
	}
	var wrap apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &wrap); err != nil {
		t.Fatalf("unmarshal version list wrapper failed: %v", err)
	}
	if wrap.Code != http.StatusOK {
		t.Fatalf("version list code=%d msg=%s", wrap.Code, wrap.Msg)
	}
	var data versionListResp
	if err := json.Unmarshal(wrap.Data, &data); err != nil {
		t.Fatalf("unmarshal version list data failed: %v", err)
	}
	if data.DocumentID != documentID || len(data.Items) < 2 {
		t.Fatalf("unexpected version list data: %+v", data)
	}
	var hasV1, hasV2, hasV2Signer bool
	for i := range data.Items {
		item := data.Items[i]
		if item.VersionNo == 1 && item.DisplayVersion == "v1.0" && item.ActionType == model.DocumentVersionActionInitial {
			hasV1 = true
		}
		if item.VersionNo == 2 && item.DisplayVersion == "v2.0" && item.ActionType == model.DocumentVersionActionSignApplied {
			hasV2 = true
			if item.SignerID == signerID && strings.TrimSpace(item.SignerName) != "" {
				hasV2Signer = true
			}
		}
	}
	if !hasV1 || !hasV2 || !hasV2Signer {
		t.Fatalf("version list missing expected records: hasV1=%v hasV2=%v hasV2Signer=%v data=%+v", hasV1, hasV2, hasV2Signer, data.Items)
	}
}

func assertVersionPreviewAPI(t *testing.T, engine http.Handler, documentID uint, documentVersion int) {
	t.Helper()
	rows, err := dao.DocumentVersionDao.SelectByDocumentID(documentID)
	if err != nil {
		t.Fatalf("select document versions failed: %v", err)
	}
	var targetID uint
	for i := range rows {
		if rows[i].VersionNo == documentVersion && rows[i].ActionType == model.DocumentVersionActionSignApplied {
			targetID = rows[i].ID
			break
		}
	}
	if targetID == 0 {
		t.Fatalf("cannot find target version id for versionNo=%d", documentVersion)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/document-versions/"+uintToString(targetID)+"/preview", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("preview version status=%d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(strings.ToLower(ct), "application/pdf") {
		t.Fatalf("preview version content-type should be application/pdf, got %s", ct)
	}
}
