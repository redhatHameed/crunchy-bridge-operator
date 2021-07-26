/*
 * Copyright (C) 2020 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dbaasredhatcom

import (
	"context"
	"fmt"
	dbaasoperator "github.com/RHEcosystemAppEng/dbaas-operator/api/v1alpha1"
	"github.com/go-logr/logr"

	v1 "k8s.io/api/apps/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	label "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/pointer"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"
)

const (
	PROVIDER           = "Red Hat DBaaS / Crunchy Bridge"
	DISPLAYNAME        = "Crunchy Bridge managed PostgreSQL"
	DISPLAYDESCRIPTION = " The Crunchy Bridge Fully Managed Postgres as a Service."

	INVENTORYDATAVALUE     = "CrunchyBridgeInventory"
	CONNECTIONDATAVALUE    = "CrunchyBridgeConnection"
	NAME                   = "crunchy-bridge-registration"
	ICONDATA               = "PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0idXRmLTgiPz4KPHN2ZyB2ZXJzaW9uPSIxLjEiIGlkPSJMYXllcl8xIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHhtbG5zOnhsaW5rPSJodHRwOi8vd3d3LnczLm9yZy8xOTk5L3hsaW5rIiB4PSIwcHgiIHk9IjBweCIKCSB2aWV3Qm94PSIwIDAgMzYzLjg1IDM2My4wNSIgc3R5bGU9ImVuYWJsZS1iYWNrZ3JvdW5kOm5ldyAwIDAgMzYzLjg1IDM2My4wNTsiIHhtbDpzcGFjZT0icHJlc2VydmUiPgo8c3R5bGUgdHlwZT0idGV4dC9jc3MiPgoJLnN0MHtmaWxsOiNGRkZGRkY7fQoJLnN0MXtmaWxsOnVybCgjU1ZHSURfMV8pO30KCS5zdDJ7ZmlsbDojNDE0MTQxO30KCS5zdDN7Y2xpcC1wYXRoOnVybCgjU1ZHSURfM18pO30KCS5zdDR7ZmlsbDpub25lO30KCS5zdDV7ZmlsbDp1cmwoI1NWR0lEXzRfKTt9Cgkuc3Q2e2ZpbGw6IzQ2NkJCMjt9Cgkuc3Q3e2ZpbGw6IzFDNDQ5Qjt9Cjwvc3R5bGU+CjxnPgoJPHBhdGggY2xhc3M9InN0MCIgZD0iTTM1My44LDIwOS41M2MtMC4yMy0zLjY1LDEuOTUtNy4wOCw1LjAxLTguOTVjMC4xMS01LjYxLDAuMzctMTEuMjIsMC40My0xNi44NGMwLjA0LTQuMS0wLjA1LTguMi0wLjQ0LTEyLjI5CgkJYy0wLjA4LTAuODktMC4yLTEuNzctMC4yOS0yLjY2Yy0wLjAzLTAuMjEtMC4wNS0wLjM3LTAuMDctMC41MWMtMC4wNy0wLjQxLTAuMTQtMC44My0wLjIyLTEuMjRjLTAuNDEtMi4xNS0wLjkyLTQuMjctMS41LTYuMzgKCQljLTAuNTEtMS44NC0wLjUtMy42Ni0wLjA3LTUuMzNjLTAuNjktMS4zNy0xLjE1LTIuODgtMS4zNS00LjRjLTAuMDYtMC40OC0wLjEzLTAuOTYtMC4yLTEuNDRjLTAuMDItMC4xMy0wLjA1LTAuMjktMC4wOC0wLjQ5CgkJYy0wLjIyLTEuMjMtMC40My0yLjQ2LTAuNjctMy42OGMtMC40Mi0yLjE0LTAuODgtNC4yNi0xLjQtNi4zOGMtMS4xLTQuNTUtMi40Mi05LjA1LTMuODktMTMuNDljLTEuNDQtNC4zNC0yLjktOC43LTQuNTItMTIuOTgKCQljLTAuNzgtMi4wNi0xLjYxLTQuMDktMi40Ny02LjExYy0wLjE0LTAuMy0wLjQ2LTEuMDMtMC42Mi0xLjM3Yy0wLjUxLTEuMDgtMS4wMi0yLjE2LTEuNTYtMy4yM2MtMi4wMS00LjAzLTQuMjMtNy45Ny02LjY1LTExLjc2CgkJYy0xLjExLTEuNzMtMi4yMi0zLjQ3LTMuMzYtNS4xOGMtMC42Mi0wLjkzLTEuMjctMS44NS0xLjk1LTIuNzRjLTAuMjEtMC4yOC0wLjQzLTAuNTYtMC42NS0wLjg0Yy0wLjA4LTAuMS0wLjE4LTAuMjItMC4zMS0wLjM3CgkJYy00Ljg0LTUuNTYtOS45OS0xMC44OC0xNC41Mi0xNi43MWMtMi4yLTAuMzEtNC4zNi0xLjI2LTYuMTEtMi42OGMtMS4xMy0wLjkxLTIuMjYtMS44NS0zLjQ0LTIuNzFjLTAuMjQtMC4xNy0wLjU2LTAuMzktMC44My0wLjU4CgkJYy0wLjQ0LTAuMzItMC44OC0wLjYzLTEuMzItMC45NGMtMi4zNS0xLjY1LTQuNzUtMy4yMy03LjE5LTQuNzVjLTQuOTgtMy4xMS0xMC4xMi01Ljk4LTE1LjMzLTguNjgKCQljLTEwLjUyLTUuNDUtMjEuMzctMTAuMjItMzIuMjktMTQuNzljLTQuNjItMS45My05LjI1LTMuODMtMTMuODgtNS43NGMtMC4zMi0wLjEzLTEuMjMtMC40OS0xLjQ2LTAuNTgKCQljLTAuNTUtMC4yMS0xLjExLTAuNDItMS42Ny0wLjYzYy0xLjMyLTAuNDgtMi42NC0wLjk1LTMuOTgtMS4zOWMtMi42My0wLjg2LTUuMjgtMS42NC03Ljk2LTIuMzNjLTUuNTUtMS40My0xMS4xOC0yLjQzLTE2Ljg2LTMuMTgKCQljLTAuMTMtMC4wMi0wLjI0LTAuMDMtMC4zMy0wLjA1Yy0wLjEzLTAuMDEtMC4yOS0wLjAyLTAuNDktMC4wNGMtMC41OS0wLjA1LTEuMTktMC4xMi0xLjc4LTAuMTdjLTEuMzQtMC4xMi0yLjY4LTAuMjMtNC4wMi0wLjMxCgkJYy0yLjY4LTAuMTgtNS4zNy0wLjI5LTguMDYtMC4zNWMtMTEuMzgtMC4yNi0yMi43NSwwLjQ1LTM0LjA4LDEuNDRjLTMsMC4yNi02LjAyLDAuNS05LjAxLDAuODZjLTAuNDgsMC4wNi0xLjA3LDAuMDgtMS41NiwwLjE5CgkJYy0wLjExLDAuMDItMC4yMSwwLjA1LTAuMywwLjA2Yy0xLjIzLDAuMjEtMi40NiwwLjQ1LTMuNjksMC43MWMtNS4xLDEuMDgtMTAuMTEsMi41OC0xNSw0LjRjLTEuMjgsMC40OC0yLjU1LDAuOTgtMy44MSwxLjQ5CgkJYy0wLjUsMC4yMS0xLjAxLDAuNDMtMS41MSwwLjYzYy0wLjA5LDAuMDQtMC4xNywwLjA3LTAuMjQsMC4xYy0yLjIyLDEuMDMtNC40MywyLjA3LTYuNiwzLjE4Yy00LjUyLDIuMzEtOC45NCw0Ljg0LTEzLjI3LDcuNDgKCQljLTguOTQsNS40Ni0xNy41NSwxMS40NC0yNi4yLDE3LjMzYy00LjMxLDIuOTQtOC43MSw1Ljc0LTEzLjAzLDguNjZjLTIuMjEsMS40OS00LjM5LDMuMDMtNi41Miw0LjY0YzAsMC0wLjQ5LDAuMzgtMC44NywwLjY3CgkJYy0wLjM4LDAuMzEtMC45OSwwLjgxLTEsMC44MmMtMC44NSwwLjcyLTEuNjksMS40Ni0yLjUsMi4yMWMtMS4xMywxLjA1LTIuMjMsMi4xMy0zLjI5LDMuMjVjLTAuNDgsMC41MS0wLjkzLDEuMDctMS40MywxLjU2CgkJYy0wLjEzLDAuMTMtMC40NCwwLjQ5LTAuNTcsMC42NGMtMS40MiwxLjk3LTIuNjUsNC4xMi0zLjksNi4xOWMtMy4yMSw1LjMzLTYuNCwxMC41NC0xMC4xNiwxNS41Yy0xLjgyLDIuNDEtNC43MSwzLjI3LTcuNTgsMy4wNgoJCUM2Ljk2LDEyMy45MiwwLDE1MS42NywwLDE4MS4wNWMwLDEwMC41Miw4MS40OCwxODIsMTgyLDE4MmM4Ny41OSwwLDE2MC43Mi02MS44NywxNzguMDktMTQ0LjI5CgkJQzM1Ni41LDIxNy4yMSwzNTQuMDYsMjEzLjcsMzUzLjgsMjA5LjUzeiIvPgoJPGc+CgkJPGc+CgkJCTxsaW5lYXJHcmFkaWVudCBpZD0iU1ZHSURfMV8iIGdyYWRpZW50VW5pdHM9InVzZXJTcGFjZU9uVXNlIiB4MT0iMTgxLjkyNjMiIHkxPSIwIiB4Mj0iMTgxLjkyNjMiIHkyPSIzMzUuNDA2OCI+CgkJCQk8c3RvcCAgb2Zmc2V0PSIwIiBzdHlsZT0ic3RvcC1jb2xvcjojMjU5Q0Q3Ii8+CgkJCQk8c3RvcCAgb2Zmc2V0PSIxIiBzdHlsZT0ic3RvcC1jb2xvcjojMDY2OEIyIi8+CgkJCTwvbGluZWFyR3JhZGllbnQ+CgkJCTxwYXRoIGNsYXNzPSJzdDEiIGQ9Ik02OS43OSwyOTAuNjdjLTAuNy0zLjg1LTEuMTUtNy43MS0xLjI4LTExLjU5Yy0wLjEzLTMuODcsMC4wNS03Ljc2LDAuNjEtMTEuNjYKCQkJCWMwLjc1LTUuMTYsMi40OS05LjgyLDQuODUtMTQuMTZjMi4zNS00LjM0LDUuMzItOC4zOCw4LjUtMTIuMzFjMC43LTAuODYsMS40MS0xLjcxLDIuMTEtMi41N2MwLjA2LTAuMDcsMC4wNS0wLjE5LDAuMTQtMC42MQoJCQkJYy04LjIyLTEuMDUtMTQuMTctNS4wMy0xNy4zLTEyLjY0Yy0xLjM1LTMuMjctMy43NC00LjY0LTYuNTYtNS4zM2MtMC45NC0wLjIzLTEuOTMtMC4zOC0yLjk0LTAuNWMtMS43MS0wLjItMy40NS0wLjEyLTUuMTQtMC4zOAoJCQkJYy0zLjI0LTAuNS01LjQ1LTIuMjQtNi4zNS00LjkzYy0wLjMtMC45LTAuNDUtMS45LTAuNDUtM2MwLjAxLTEuNi0wLjA5LTIuNjgtMi4wMy0yLjk5Yy0yLjk5LTAuNDgtNS40LTEuNS03LjMxLTIuODkKCQkJCWMtMC45NS0wLjY5LTEuNzgtMS40OC0yLjUtMi4zNGMtMC45MS0xLjA5LTEuNTctMi4zMy0yLjEzLTMuNjJjLTEuMTctMi43My0xLjY2LTUuNzYtMS41My04Ljc1YzAtMC4wMywwLTAuMDUsMC0wLjA4CgkJCQljMC4wNS0xLjExLDAuMTgtMi4yMSwwLjM3LTMuMjhjMS4yMS02Ljc0LDIuOTktMTMuMzgsNC40Mi0yMC4wOGMxLjE5LTUuNjIsMi4yMi0xMS4yOCwzLjM1LTE2LjkxCgkJCQljMS4yNC02LjE3LDUuMjMtMTAuNDMsMTAuMTEtMTMuODhjMS4zMy0wLjk0LDIuMzgtMS45OSwzLjE4LTMuMnMxLjM1LTIuNTcsMS42OS00LjEzYzEuMi01LjUsNC4wOS04LjYxLDguNjMtOS4zNAoJCQkJYzEuNTEtMC4yNSwzLjItMC4yMyw1LjA3LDAuMDVjMS40LDAuMjEsMi43OCwwLjU0LDQuMTksMC43YzEuNjYsMC4xOSwzLjI5LDAuMjgsNC45LDAuMjRjNC44Mi0wLjEyLDkuMzctMS4zOCwxMy41Ni00LjQzCgkJCQljNi4wNS00LjM5LDEyLjI1LTguNTksMTguMjgtMTMuMDJjMC43Ny0wLjU2LDEuMjEtMi4wMiwxLjIzLTMuMDdjMC4wOS00LjM1LTAuMi04LjctMC4wNC0xMy4wNGMwLjA5LTIuNTUsMC40OS00Ljg2LDEuMTgtNi45NAoJCQkJYzIuMDgtNi4yNCw2Ljg1LTEwLjM3LDE0LjIyLTEyLjIyYzUuMTktMS4zLDkuNDMtNC4wNywxMy4yNC03LjYzYzAuMjktMC4yOCwwLjU5LTAuNzYsMC43OC0xLjI0YzAuMTktMC40OCwwLjI3LTAuOTUsMC4xMi0xLjIxCgkJCQljLTEuNDQtMi42NC0xLjk4LTUuMTQtMS44OC03LjU1YzAuMDctMS42LDAuNDItMy4xNiwwLjk3LTQuNjljMC44My0yLjMsMi4xMi00LjUzLDMuNTktNi43M2MwLjQ1LTAuNjcsMC45Mi0xLjMyLDEuNjItMi4zMQoJCQkJYzEuMDcsMS4xNSwyLjE5LDIuMjUsMy4zLDMuMzZjMy4zMSwzLjMzLDYuNDYsNi43NSw3LjUxLDExLjk3YzYuMDgtMS4zLDEyLjI5LTIuNTQsMTguNDYtMy45N2M0LjU2LTEuMDYsOS4xMS0xLjc1LDEzLjY0LTIuMTEKCQkJCWMxMy41OS0xLjA4LDI3LjAxLDAuODYsNDAuMjQsNS4wOGMzLjcyLDEuMTgsNy4xNSwyLjksMTAuNCw0LjljMi43NCwxLjY4LDUuMzYsMy41Miw3Ljc1LDUuN2MxLjc0LDEuNTksMy4zOSwzLjMsNC45Myw1LjEyCgkJCQljMi45MSwzLjQyLDUuODEsNi44NSw4LjY4LDEwLjNjNS43NSw2LjksMTEuMzksMTMuODcsMTYuODIsMjAuOThjMi43MiwzLjU2LDUuMzgsNy4xNSw3Ljk3LDEwLjc5CgkJCQljNS4xOSw3LjI4LDEwLjExLDE0Ljc1LDE0LjY0LDIyLjQ3YzQuNTMsNy43Myw4LjY5LDE1LjcxLDEyLjM0LDI0LjA0YzQuMTQsOS40NCw3LjksMTkuMDQsMTEuOTUsMjguNTIKCQkJCWMwLjY0LDEuNSwxLjY4LDMsMi45MSw0LjAzYzEyLjQyLDEwLjQxLDI0LjA1LDIxLjU1LDM0LjU2LDMzLjY5YzIuODQtMTIuNzcsNC40Ni0yNi4wMSw0LjQ2LTM5LjY0CgkJCQljMC0zNy41MS0xMS4zNy03Mi4zNS0zMC44Mi0xMDEuMzFjLTYuNTMtOS42My0xMy45NS0xOC42Mi0yMi4xNS0yNi44MmMtMjAuNjUtMjAuNjUtNDYuMjEtMzYuMzgtNzQuNzUtNDUuMjgKCQkJCUMyMTguOTksMi44OCwyMDAuNzgsMCwxODEuOTIsMEMxMjUuMzQsMCw3NC42OSwyNS44OSw0MS4yLDY2LjQ2Yy0zLjY3LDQuNDQtNy4xMiw5LjA3LTEwLjM2LDEzLjg1CgkJCQljLTE5LjQ2LDI4Ljk2LTMwLjgzLDYzLjgtMzAuODMsMTAxLjMxYzAsNjQuODIsMzQsMTIxLjU2LDg1LjA0LDE1My43OGMtMy4zMi02Ljc5LTYuMzMtMTMuNjktOC44NC0yMC44CgkJCQlDNzMuNDcsMzA2Ljg2LDcxLjI4LDI5OC45LDY5Ljc5LDI5MC42N3oiLz4KCQkJPGc+CgkJCQk8cGF0aCBjbGFzcz0ic3QyIiBkPSJNNzguNTcsMTQ4LjJsLTAuMTQsMC43N2MwLDAsMy40NSwyMC4zNSwzNy4xNiw3Ljc2Yy0yLjExLTEuNTctMy4yMy00LjAyLTMuMi03LjYyCgkJCQkJYy02LjUyLDUuODYtMTMuODIsNy4yLTIxLjYxLDYuMzFDODUuMTEsMTU0Ljc4LDgwLjk3LDE1MS45Nyw3OC41NywxNDguMnoiLz4KCQkJCTxwYXRoIGNsYXNzPSJzdDIiIGQ9Ik04Mi45NywxMjYuMTlsMC4wOCwwLjAxYzAuMDEtMC4wMSwwLjAxLTAuMDIsMC4wMi0wLjAzQzgzLjAzLDEyNi4xOCw4MywxMjYuMTksODIuOTcsMTI2LjE5eiIvPgoJCQkJPHBhdGggY2xhc3M9InN0MiIgZD0iTTk3LjEsMTQ3Ljk1YzIuNDUtMi4yMSwyLjg2LTYuNSwxLTkuODFjLTEuNDUtMi41OS00LjQxLTMuNjgtNy42NC0yLjU1Yy0yLjMzLDAuODEtNC4yOCwyLjEzLTMuOTMsNS4xNwoJCQkJCWMxLjUzLTAuMjIsMi44Ny0wLjQsNC42OS0wLjY2Yy0xLjQxLDMuMTctMC40NSw1LjMxLDIuMjMsNy4wMWMtMS4xNSwwLjI2LTEuODIsMC40MS0yLjksMC42NQoJCQkJCUM5MywxNDkuODcsOTUuMDgsMTQ5Ljc4LDk3LjEsMTQ3Ljk1eiIvPgoJCQkJPHBhdGggY2xhc3M9InN0MiIgZD0iTTEyMi43OSwxMjAuMThjLTUuNDMsMi41NC0xMC43OSwzLjc2LTE2LjkxLDEuODVjLTMuMzItMS4wMy03LjIyLTAuMTgtMTAuODYtMC4xOAoJCQkJCWM2LjU2LDEuMjEsMTIsMy43OSwxNS4xNiw5LjMzYzUuMDgtNC4wNCwxMC4wOC04LjAzLDE1LjE4LTEyLjA5QzEyNC40MiwxMTkuNDgsMTIzLjU5LDExOS44MSwxMjIuNzksMTIwLjE4eiIvPgoJCQkJPHBhdGggY2xhc3M9InN0MiIgZD0iTTMyNC44MywxODcuNThjLTEuMjQtMS4wNC0yLjI3LTIuNTQtMi45MS00LjAzYy00LjA1LTkuNDgtNy44LTE5LjA4LTExLjk1LTI4LjUyCgkJCQkJYy0zLjY2LTguMzItNy44MS0xNi4zMS0xMi4zNC0yNC4wNGMtNC41My03LjczLTkuNDUtMTUuMTktMTQuNjQtMjIuNDdjLTIuNTktMy42NC01LjI2LTcuMjQtNy45Ny0xMC43OQoJCQkJCWMtNS40My03LjEyLTExLjA4LTE0LjA4LTE2LjgyLTIwLjk4Yy0yLjg3LTMuNDUtNS43Ny02Ljg4LTguNjgtMTAuM2MtMS41NS0xLjgyLTMuMTktMy41My00LjkzLTUuMTIKCQkJCQljLTIuMzktMi4xNy01LjAxLTQuMDItNy43NS01LjdjLTMuMjUtMi02LjY4LTMuNzEtMTAuNC00LjljLTEzLjIzLTQuMjEtMjYuNjUtNi4xNS00MC4yNC01LjA4Yy00LjUzLDAuMzYtOS4wOCwxLjA1LTEzLjY0LDIuMTEKCQkJCQljLTYuMTcsMS40My0xMi4zNywyLjY3LTE4LjQ2LDMuOTdjLTEuMDYtNS4yMi00LjItOC42NC03LjUxLTExLjk3Yy0xLjEtMS4xMS0yLjIzLTIuMjEtMy4zLTMuMzZjLTAuNywwLjk5LTEuMTcsMS42NC0xLjYyLDIuMzEKCQkJCQljLTEuNDcsMi4yMS0yLjc2LDQuNDQtMy41OSw2LjczYy0wLjU1LDEuNTMtMC45MSwzLjA5LTAuOTcsNC42OWMtMC4xLDIuNCwwLjQzLDQuOSwxLjg4LDcuNTVjMC4xNCwwLjI2LDAuMDYsMC43My0wLjEyLDEuMjEKCQkJCQljLTAuMTksMC40OC0wLjQ4LDAuOTYtMC43OCwxLjI0Yy0zLjgsMy41Ni04LjA1LDYuMzMtMTMuMjQsNy42M2MtNy4zNywxLjg1LTEyLjE0LDUuOTgtMTQuMjIsMTIuMjIKCQkJCQljLTAuNjksMi4wOC0xLjA5LDQuNC0xLjE4LDYuOTRjLTAuMTYsNC4zNCwwLjE0LDguNywwLjA0LDEzLjA0Yy0wLjAyLDEuMDUtMC40NywyLjUxLTEuMjMsMy4wNwoJCQkJCWMtNi4wMyw0LjQzLTEyLjIyLDguNjMtMTguMjgsMTMuMDJjLTQuMTksMy4wNC04Ljc0LDQuMzEtMTMuNTYsNC40M2MtMS42MSwwLjA0LTMuMjQtMC4wNS00LjktMC4yNAoJCQkJCWMtMS40LTAuMTYtMi43OS0wLjUtNC4xOC0wLjdjLTEuODctMC4yOC0zLjU2LTAuMy01LjA3LTAuMDVjLTQuNTMsMC43NC03LjQyLDMuODQtOC42Myw5LjM0Yy0wLjM0LDEuNTctMC44OSwyLjkzLTEuNjksNC4xMwoJCQkJCWMtMC44LDEuMjEtMS44NCwyLjI2LTMuMTgsMy4yYy00Ljg4LDMuNDUtOC44Nyw3LjcxLTEwLjExLDEzLjg4Yy0xLjEzLDUuNjMtMi4xNiwxMS4yOS0zLjM1LDE2LjkxCgkJCQkJYy0xLjQyLDYuNzEtMy4yLDEzLjM0LTQuNDIsMjAuMDhjLTAuMTksMS4wNy0wLjMyLDIuMTctMC4zNywzLjI4YzAsMC4wMywwLDAuMDUsMCwwLjA4Yy0wLjEzLDIuOTksMC4zNSw2LjAzLDEuNTMsOC43NQoJCQkJCWMwLjU2LDEuMjksMS4yMiwyLjUzLDIuMTMsMy42MmMwLjcxLDAuODYsMS41NCwxLjY1LDIuNSwyLjM0YzEuOTEsMS4zOSw0LjMyLDIuNCw3LjMxLDIuODljMS45NCwwLjMxLDIuMDQsMS4zOCwyLjAzLDIuOTkKCQkJCQljMCwxLjEsMC4xNSwyLjEsMC40NSwzYzAuOSwyLjY5LDMuMTEsNC40Myw2LjM1LDQuOTNjMS42OSwwLjI2LDMuNDQsMC4xOCw1LjE0LDAuMzhjMS4wMSwwLjEyLDIsMC4yNywyLjk0LDAuNQoJCQkJCWMyLjgyLDAuNjgsNS4yMSwyLjA2LDYuNTUsNS4zM2MzLjEzLDcuNiw5LjA4LDExLjU4LDE3LjMsMTIuNjRjLTAuMDksMC40MS0wLjA4LDAuNTQtMC4xNCwwLjYxYy0wLjcsMC44Ni0xLjQxLDEuNzEtMi4xMSwyLjU3CgkJCQkJYy0zLjE5LDMuOTMtNi4xNSw3Ljk2LTguNSwxMi4zMWMtMi4zNSw0LjM1LTQuMSw5LTQuODUsMTQuMTZjLTAuNTcsMy45LTAuNzUsNy43OS0wLjYxLDExLjY2YzAuMTMsMy44OCwwLjU4LDcuNzQsMS4yOCwxMS41OQoJCQkJCWMxLjQ5LDguMjMsMy42NywxNi4xOSw2LjQxLDIzLjk0YzIuNTEsNy4xLDUuNTIsMTQuMDEsOC44NCwyMC44YzYuMjUsMy45NCwxMi43NSw3LjQ5LDE5LjQ4LDEwLjY2CgkJCQkJYzIuMTYtOC42NSw0Ljc0LTE3LjI0LDcuODktMjUuNzNjMS44LTQuODcsNC40NS05LjU1LDcuMzYtMTMuODdjNS44Ny04LjcxLDE0LjIxLTEyLjkzLDI0Ljg3LTEyLjE3CgkJCQkJYzIzLjI4LDEuNjUsNDUuMzMsNy41LDY2LjEyLDE4LjMzYzE3LjAyLDguODYsMjcuNzMsMjIuMSwzMi44NSwzOS45NmM4LTIuODksMTUuNzktNi4yLDIzLjIyLTEwLjEzCgkJCQkJYy0wLjA1LTAuMTMtMC4xMi0wLjIzLTAuMTUtMC4zOGMtMi42Ny0xMy40MywxLjEyLTM2LjE1LDE3LjQyLTQ1Ljg1YzkuNzctNS44MSwyMC4zNy0xMC4yMywzMC42NC0xNS4xOAoJCQkJCWMyLjY5LTEuMjksNS41Ny0yLjE5LDguNzQtMy40MmMtMi40MS0xLjM3LTQuNS0yLjU0LTYuNTctMy43NWMtMTAuNTktNi4xOC0yMC41NC0xMy4yMS0yOS44Ny0yMS4yNAoJCQkJCWMtMTMuMDMtMTEuMjEtMTUuNzEtMjUuMDMtMTEuNTYtNDAuODljMC44MS0zLjEsMi02LjEsMy4wOC05LjM0Yy0wLjczLDAuNTMtMS4zMywwLjk2LTIuNTEsMS44MQoJCQkJCWM0LjM5LTEwLjQ3LDUuOC0yMC44Myw1LjY1LTMxLjQxYy0wLjA2LTQuNTQtMC43OS05LjA3LTAuODUtMTMuNjFjLTAuMDYtMy43MiwwLjQxLTcuNDUsMC42Ni0xMS4xN2MwLjEzLTIsMC4yOC0zLjk5LDAuNDUtNi40OAoJCQkJCWMtNC45MSwxLjItNy45OC0wLjkxLTkuNjktNC44OGMwLjc5LDcuMjQsMi4xNywxNC40MiwyLjIxLDIxLjYxYzAuMDUsOS41LTMuOSwxOC4xNy04LjI2LDI2LjM2CgkJCQkJYy04Ljg3LDE2LjY4LTIyLjk5LDI3LjkzLTM5LjU2LDM2LjM2Yy02LjQ2LDMuMjktMTIuNzMsMi4zOS0xOC44Mi0xLjA0Yy0yLjExLTEuMTktNC4xNS0yLjUyLTYuMjctMy42OQoJCQkJCWMtMS45Ny0xLjA5LTIuOTEtMi4zMi0yLjIyLTQuODNjMC42OC0yLjQ4LDIuMTYtMy4xOSw0LjM3LTMuMzFjMC4xLTAuMDEsMC4yLTAuMDQsMC4zLTAuMDJjNi4zNCwxLjA1LDkuNTgtMi45MSwxMS44My03Ljc5CgkJCQkJYzIuMTgtNC43NS0wLjA0LTktMi40NS0xMy4zYy0yLjA3LDAuNTgtNC4wNywwLjk3LTUuOTEsMS43NGMtMC44NywwLjM2LTEuNjEsMS4zNy0yLjA5LDIuMjVjLTMuNzEsNi44My03LjExLDEzLjgzLTExLjEyLDIwLjQ3CgkJCQkJYy0xNy4xNywyNi41Mi00NS40MSwxOC4xMi00NS40MSwxOC4xMmMtMTUuMjYtMi42Ny0zNy43OC0wLjc2LTM3Ljc4LTAuNzZjLTguNzgsMS41My0yNC40MiwxLjkxLTI0LjQyLDEuOTEKCQkJCQljLTExLjgzLDAtMTQuODgtMTIuMzQtMTQuODgtMTIuMzRjNi42MS00LjMyLDQ2LjA0LDEuNzgsNDYuMDQsMS43OGMyNi4wNCwzLjc1LDQzLjUyLTYuMTIsNTAuMTMtMTAuNzcKCQkJCQljLTM0Ljc0LDE2LjA4LTc2LjUyLTYuMjktMTEyLjYsMi45Yy0zLjU0LDAuOS01LjA0LTQuNTYtMS41MS01LjQ2YzI5LjMxLTcuNDYsNjMuNDQsNi40Miw5My40MSwyLjY3CgkJCQkJYzEwLjk5LTIuMDUsMjEuMjYtNi41MSwzMC44LTEyLjU4YzMuOTktMi41NCw2LjE2LTYuMzgsNi4xNC0xMS4yM2MtMC4wMi00LjM0LDAuMzItOC43Ni0wLjM0LTEzLjAyCgkJCQkJYy0xLjk5LTEyLjktMTYuNTgtMjQuMDMtMjkuOTMtMjMuMDljMCwwLDIwLjI1LDEyLjIsMjEuODgsMjMuMzRjNS41NywyMS43MS0zMS41OCwyNi42NS0zMS41OCwyNi42NXMtMjEuOTgsNC44OS00OC41OS0zLjkxCgkJCQkJYzAsMC0xOS4yMi00LjMzLTQ1LjM4LDMuMDZjMCwwLTkuMTIsMy4zOC0xMC4yNi00Ljg0Yy0wLjgxLTUuODQsMS4wNy0xOS44LDcuNTQtMzkuM2wtMC4xLTAuMmMwLjA2LDAuMDEsMC4xMS0wLjAxLDAuMTYtMC4wMQoJCQkJCWMwLjIxLTAuNjMsMC4zOC0xLjIsMC42LTEuODRjMCwwLDIuNzUsMC45Myw2LjY4LDEuMDRjMTAuMDgtMi45OCwxNC44My0xMy4zMiwxNC44My0xMy4zMmwtOC4wNCw1LjY1bDAuNTgtMS4wNgoJCQkJCWMtMy4wNSw0LjAxLTguOTgsNS4yOC0xMy4xNywyLjMxYy0yLjYxLTEuODUtMi41NC00LjI5LDAuMzQtNi4zOWMwLjY2LTAuNDgsMS4zNi0wLjkzLDIuMDUtMS4zOGMwLjAxLDAuMjEtMC4xMiwwLjQtMC4wNywwLjYxCgkJCQkJYzIuNjksMC4zOSw3LjQ1LTAuMjUsOS4xNS0yLjU2YzEuMDgtMS40NywxLjMyLTYuNy0xLjMzLTdjMC45LTUsMy4yOC03LjAzLDkuMDUtNy4wNWMzLjg0LTAuMDIsNy42OSwwLjI5LDExLjUzLDAuMTIKCQkJCQljNi4wOS0wLjI2LDExLjctMi4yLDE2LjU4LTUuODdjMTMuNDEtMTAuMDksMjYuNzYtMjAuMjgsNDAuMTMtMzAuNDNjMC4xNC0wLjEsMC4xNy0wLjM0LDAuMy0wLjZjLTIuODktMC4xMy01Ljc2LTAuMS04LjU5LTAuNDQKCQkJCQljLTMuNTItMC40Mi02LjcxLTEuNjctOC41LTUuMDZjLTIuMDktMy45Ni0wLjQ3LTYuOSwzLjcyLTguMjhjMy43OC0xLjI1LDcuODQtMi42NSwxMC44NC01LjEyCgkJCQkJYzE4LjQyLTE1LjIsNDAuMDgtMTcuNzEsNjIuNjctMTYuNDJjNi4zNCwwLjM2LDEyLjY2LDEuNDksMTguOTEsMi43OWMtMi4wNSwwLjQ5LTExLjQ1LDIuOTYtMTQuMzYsNy41NwoJCQkJCWMtMC45OSwxLjc2LTIuNjUsMy42My0zLjAzLDUuNzRjLTAuMjcsMS40OCwxLjIyLDMuMjgsMS45Miw0Ljk0YzEuMjQtMC45NiwyLjUtMS44OSwzLjctMi44OWMwLjI4LTAuMjMsMC4zOC0wLjcxLDAuNS0xLjA5CgkJCQkJYzEuNzItNS4zNiw2LjE2LTcuOTUsMTAuODYtMTAuOThjMC40MywyLjEsMC42MSwzLjc5LDEuMTMsNS4zOGMwLjcsMi4xNi0wLjI2LDMuNTgtMS44OCw0LjYyYy0wLjgyLDAuNTMtMS45NiwwLjU2LTIuOTYsMC44MgoJCQkJCWMtMC4yLTEuMTEtMC41Ni0yLjIyLTAuNTMtMy4zMmMwLjAyLTAuOSwwLjUxLTEuNzksMC43OS0yLjY4Yy00Ljk4LDMuMjMtNi44OCw4LjItOC43MywxMy40MWMxLjE4LDAuNDIsMi4wMywwLjcyLDIuMzIsMC44MgoJCQkJCWMtMC41OCwxLjg5LTEuMTMsMy42OC0xLjcsNS41NGw0LjMxLTIuMTNjMi44Ni0xLDUuMDQtMS43Myw3LjItMi41M2MzLjM0LTEuMjQsNC45NC0zLjc3LDQuOTYtNy4yNAoJCQkJCWMwLjA0LTUuNDktMS42LTEwLjU2LTMuOTktMTUuODdjMC43OCwwLjE2LDEuNTYsMC4yOCwyLjM0LDAuNDVjNy42OSwxLjY4LDE0LjEyLDYuMDQsMTkuNTQsMTEuNjcKCQkJCQljNC4yNyw0LjQ0LDguNzgsOC44MywxMi4xNywxMy45MWM3Ljc2LDExLjYxLDE4LjQ0LDIwLjc3LDI2LjA5LDMyLjQ4YzEzLjgyLDIxLjE0LDI2LjEzLDQzLjAzLDM1LjIsNjYuNjUKCQkJCQljMC4zOSwxLjAxLDAuNzIsMi4wNSwxLjEyLDMuMThjLTIuOTQsMC45My01LjcyLDEuOC04LjQzLDIuNjVjMTcuNzgsNy4zMSwzMC4wOCwyMS4zOSw0Mi45OSwzNC41NQoJCQkJCWMzLjM2LDMuNDMsNi41Myw3LjAyLDkuNjEsMTAuNjdjMC44NC0zLjAyLDEuNjYtNi4wNiwyLjM1LTkuMTRDMzQ4Ljg3LDIwOS4xMiwzMzcuMjUsMTk3Ljk5LDMyNC44MywxODcuNTh6IE0xNjMuMTcsMjYyLjM0CgkJCQkJYzIwLjY4LDIuMzQsNDAuOTYsMS40Miw1OC42Mi0xMS41MWM3LjI0LTUuMywxMy45LTExLjM4LDIwLjg3LTE3LjA2YzEuMDYtMC44NiwyLjI3LTEuNzUsMy41NS0yLjA3CgkJCQkJYzguODYtMi4xOSwxNi4yNi02LjU0LDIxLjYyLTE0LjA4YzAuMTUtMC4yMSwwLjQzLTAuMzIsMS4yOS0wLjk0Yy0wLjg5LDIuNjItMS4xOSw0Ljg0LTIuMzEsNi41Yy0xLjgxLDIuNjgtMi4yMSw1LjUyLTIuNCw4LjU4CgkJCQkJYy0xLjA1LDE2LjczLTIuOTIsMzMuMjctMTEuNTQsNDguMThjLTAuMSwwLjE3LTAuMiwwLjM1LTAuMjgsMC41NGMtNS42LDEzLjU2LTE2LjUsMTguMzEtMzAuMywxOC4xNAoJCQkJCWMtMTAuMjktMC4xMy0yMC4yNS0yLjQtMzAuMTMtNWwtNTUuMzEtMTQuNzNjLTYuNDQtMi42OS02Ljk5LTQuMDYtNC4xMi0xMC4yM2MyLjk4LTYuNCw2LjAyLTEyLjc3LDkuMjQtMTkuNTkKCQkJCQlDMTQ1Ljk2LDI1OC45NiwxNTQuNCwyNjEuMzUsMTYzLjE3LDI2Mi4zNHogTTg3LjM1LDI0Ny40NWM4LjU5LTEwLjQ0LDguNC0xMS4wNCwyMS44MS0xMS40MmM5LjUyLTAuMjcsMTkuMDksMC45MSwyOS4yOSwxLjQ5CgkJCQkJYy0wLjg1LDEuOTktMS4yOCwzLjA5LTEuNzgsNC4xNWMtNC44MywxMC4zNC05LjczLDIwLjY2LTE0LjQ3LDMxLjA0Yy0wLjg4LDEuOTMtMS44LDIuMzQtMy45LDEuOTkKCQkJCQljLTYuMzktMS4wNi0xMi42Ny0wLjYtMTkuMDQsMi44MWMxLjYzLDAuMiwyLjg3LDAuMTEsMy45MiwwLjUzYzIuNjQsMS4wNyw1LjY2LDEuODUsNy42NywzLjY5YzMuNTksMy4yOCw0LjQ5LDYuOTMsMi40MywxMi4zMQoJCQkJCWMtNS42OSwxNC44OC0xMC4yNiwzMC4xOS0xNS4yNyw0NS4zM2MtMC4xOCwwLjU2LTAuMzYsMS4xMi0wLjU0LDEuNjdjLTAuMzIsMC4xLTAuNjQsMC4yMS0wLjk2LDAuMzIKCQkJCQljLTMuMjctNy4xNC02LjcyLTE0LjE5LTkuNzUtMjEuNDNjLTUuMTMtMTIuMjUtOC45LTI0Ljg5LTEwLjIxLTM4LjJDNzUuMjksMjY4LjcxLDc5LjA5LDI1Ny41LDg3LjM1LDI0Ny40NXoiLz4KCQkJCTxwYXRoIGNsYXNzPSJzdDIiIGQ9Ik0xNTcuNTYsMTAxLjljMi45MS0yLjcxLDUuNDMtNC4yNyw3LjA0LTUuMWMtMC4zOSwyLjEzLDEuNyw1LjY5LDUuOTUsNi4yMmM3Ljk2LDAuMTEsNy44OC05LjIzLDcuODgtOS4yMwoJCQkJCWMyLjA2LTAuNTUsNC40Ni0wLjc5LDQuNDYtMC43OUMxNjYuMTksODUuMDIsMTU3LjU2LDEwMS45LDE1Ny41NiwxMDEuOXoiLz4KCQkJPC9nPgoJCTwvZz4KCTwvZz4KPC9nPgo8L3N2Zz4K"
	MEDIATYPE              = "image/svg+xml"
	KEYFIELDNAME           = "publicApiKey"
	KEYFIELDDISPLAYNAME    = "Public API Key"
	SECRETFIELDNAME        = "privateApiSecret"
	SECRETFIELDDISPLAYNAME = "Private API Secret"
	RELATEDTOLABELNAME     = "related-to"
	RELATEDTOLABELVALUE    = "dbaas-operator"
	TYPELABELNAME          = "type"
	TYPELABELVALUE         = "dbaas-provider-registration"
	DBAASPROVIDERKIND      = "DBaaSProvider"
)

var labels = map[string]string{RELATEDTOLABELNAME: RELATEDTOLABELVALUE, TYPELABELNAME: TYPELABELVALUE}

type DBaaSProviderReconciler struct {
	client.Client
	*runtime.Scheme
	Log                      logr.Logger
	Clientset                *kubernetes.Clientset
	operatorNameVersion      string
	operatorInstallNamespace string
}

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;create;update;delete;watch
// +kubebuilder:rbac:groups=dbaas.redhat.com,resources=*,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dbaas.redhat.com,resources=*/status,verbs=get;update;patch

func (r *DBaaSProviderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := r.Log.WithValues("during", "DBaaSProvider Reconciler")

	// due to predicate filtering, we'll only reconcile this operator's own deployment when it's seen the first time
	// meaning we have a reconcile entry-point on operator start-up, so now we can create a cluster-scoped resource
	// owned by the operator's ClusterRole to ensure cleanup on uninstall

	dep := &v1.Deployment{}
	if err := r.Get(ctx, req.NamespacedName, dep); err != nil {
		if errors.IsNotFound(err) {
			// CR deleted since request queued, child objects getting GC'd, no requeue
			log.Info("deployment not found, deleted, no requeue")
			return ctrl.Result{}, nil
		}
		// error fetching deployment, requeue and try again
		log.Error(err, "error fetching Deployment CR")
		return ctrl.Result{}, err
	}

	isCrdInstalled, err := r.checkCrdInstalled(dbaasoperator.GroupVersion.String(), DBAASPROVIDERKIND)
	if err != nil {
		log.Error(err, "error discovering GVK")
		return ctrl.Result{}, err
	}
	if !isCrdInstalled {
		log.Info("CRD not found, requeueing with rate limiter")
		// returning with 'Requeue: true' will invoke our custom rate limiter seen in SetupWithManager below
		return ctrl.Result{Requeue: true}, nil
	}

	instance := &dbaasoperator.DBaaSProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: NAME,
		},
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(instance), instance); err != nil {
		if errors.IsNotFound(err) {
			// CR deleted since request queued, child objects getting GC'd, no requeue
			log.Info("resource not found, creating now")

			// crunchy bridge registration custom resource isn't present,so create now with ClusterRole owner for GC
			opts := &client.ListOptions{
				LabelSelector: label.SelectorFromSet(map[string]string{
					"olm.owner":      r.operatorNameVersion,
					"olm.owner.kind": "ClusterServiceVersion",
				}),
			}
			clusterRoleList := &rbac.ClusterRoleList{}
			if err := r.List(context.Background(), clusterRoleList, opts); err != nil {
				log.Error(err, "unable to list ClusterRoles to seek potential operand owners")
				return ctrl.Result{}, err
			}

			if len(clusterRoleList.Items) < 1 {
				err := errors.NewNotFound(
					schema.GroupResource{Group: "rbac.authorization.k8s.io", Resource: "ClusterRole"}, "potentialOwner")
				log.Error(err, "could not find ClusterRole owned by CSV to inherit operand")
				return ctrl.Result{}, err
			}

			instance = bridgeProviderCR(clusterRoleList)
			if err := r.Create(ctx, instance); err != nil {
				log.Error(err, "error while creating new cluster-scoped resource")
				return ctrl.Result{}, err
			} else {
				log.Info("cluster-scoped resource created")
				return ctrl.Result{}, nil
			}
		}
		// error fetching the resource, requeue and try again
		log.Error(err, "error fetching the resource")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// bridgeProviderCR CR for crunchy bridge registration
func bridgeProviderCR(clusterRoleList *rbac.ClusterRoleList) *dbaasoperator.DBaaSProvider {
	instance := &dbaasoperator.DBaaSProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: NAME,
			OwnerReferences: []metav1.OwnerReference{
				{

					APIVersion:         "rbac.authorization.k8s.io/v1",
					Kind:               "ClusterRole",
					UID:                clusterRoleList.Items[0].GetUID(), // doesn't really matter which 'item' we use
					Name:               clusterRoleList.Items[0].Name,
					Controller:         pointer.BoolPtr(true),
					BlockOwnerDeletion: pointer.BoolPtr(false),
				},
			},
			Labels: labels,
		},

		Spec: dbaasoperator.DBaaSProviderSpec{
			Provider: dbaasoperator.DatabaseProvider{
				Name:               PROVIDER,
				DisplayName:        DISPLAYNAME,
				DisplayDescription: DISPLAYDESCRIPTION,
				Icon: dbaasoperator.ProviderIcon{
					Data:      ICONDATA,
					MediaType: MEDIATYPE,
				},
			},
			InventoryKind:  INVENTORYDATAVALUE,
			ConnectionKind: CONNECTIONDATAVALUE,
			CredentialFields: []dbaasoperator.CredentialField{
				{
					Key:         KEYFIELDNAME,
					DisplayName: KEYFIELDDISPLAYNAME,
					Type:        "string",
					Required:    true,
				},
				{
					Key:         SECRETFIELDNAME,
					DisplayName: SECRETFIELDDISPLAYNAME,
					Type:        "maskedstring",
					Required:    true,
				},
			},
		},
	}
	return instance
}

// CheckCrdInstalled checks whether dbaas provider CRD, has been created yet
func (r *DBaaSProviderReconciler) checkCrdInstalled(groupVersion, kind string) (bool, error) {
	resources, err := r.Clientset.Discovery().ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	for _, r := range resources.APIResources {
		if r.Kind == kind {
			return true, nil
		}
	}
	return false, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DBaaSProviderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	log := r.Log.WithValues("during", "DBaaSProviderReconciler setup")

	// envVar set in controller-manager's Deployment YAML
	if operatorInstallNamespace, found := os.LookupEnv("INSTALL_NAMESPACE"); !found {
		err := fmt.Errorf("INSTALL_NAMESPACE must be set")
		log.Error(err, "error fetching envVar")
		return err
	} else {
		r.operatorInstallNamespace = operatorInstallNamespace
	}

	// envVar set for all operators
	if operatorNameEnvVar, found := os.LookupEnv("OPERATOR_CONDITION_NAME"); !found {
		err := fmt.Errorf("OPERATOR_CONDITION_NAME must be set")
		log.Error(err, "error fetching envVar")
		return err
	} else {
		r.operatorNameVersion = operatorNameEnvVar
	}

	customRateLimiter := workqueue.NewItemExponentialFailureRateLimiter(30*time.Second, 30*time.Minute)

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{RateLimiter: customRateLimiter}).
		For(&v1.Deployment{}).
		WithEventFilter(r.ignoreOtherDeployments()).
		Complete(r)
}

//ignoreOtherDeployments  only on a 'create' event is issued for the deployment
func (r *DBaaSProviderReconciler) ignoreOtherDeployments() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return r.evaluatePredicateObject(e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func (r *DBaaSProviderReconciler) evaluatePredicateObject(obj client.Object) bool {
	lbls := obj.GetLabels()
	if obj.GetNamespace() == r.operatorInstallNamespace {
		if val, keyFound := lbls["olm.owner.kind"]; keyFound {
			if val == "ClusterServiceVersion" {
				if val, keyFound := lbls["olm.owner"]; keyFound {
					return val == r.operatorNameVersion
				}
			}
		}
	}
	return false
}
