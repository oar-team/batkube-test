library(ggplot2)
library(dplyr)
library(patchwork)
library(tidyr)

compute.metrics <- function(filename) {
	data <- read.csv(filename)
	makespan <- max(data$finish_time)
	mean_waiting_time <- mean(data$waiting_time)
	return(list("makespan" = makespan, "mwt" = mean_waiting_time))
}

simu.burst.makespan <- c()
simu.burst.mwt <- c()
simu.spaced.makespan <- c()
simu.spaced.mwt <- c()
simu.realistic.makespan <- c()
simu.realistic.mwt <- c()
for (i in 0:19) {
	met <- compute.metrics(paste("../results/200_delay170_simu_", i, "_jobs.csv", sep=""))
	simu.burst.makespan <- c(simu.burst.makespan, met$makespan)
	simu.burst.mwt <- c(simu.burst.mwt, met$mwt)

	met <- compute.metrics(paste("../results/spaced_200_delay170_simu_", i, "_jobs.csv", sep=""))
	simu.spaced.makespan <- c(simu.spaced.makespan, met$makespan)
	simu.spaced.mwt <- c(simu.spaced.mwt, met$mwt)

	met <- compute.metrics(paste("../results/KIT_10h_80_simu_", i, "_jobs.csv", sep=""))
	simu.realistic.makespan <- c(simu.realistic.makespan, met$makespan)
	simu.realistic.mwt <- c(simu.realistic.mwt, met$mwt)
}
simu.burst <- data.frame(makespan=simu.burst.makespan, mwt=simu.burst.mwt)
simu.burst$type <- "burst"
simu.spaced <- data.frame(makespan=simu.spaced.makespan, mwt=simu.spaced.mwt)
simu.spaced$makespan <- simu.spaced.makespan
simu.spaced$type <- "spaced"
simu.realistic <- data.frame(makespan=simu.realistic.makespan, mwt=simu.realistic.mwt)
simu.realistic$makespan <- simu.realistic.makespan
simu.realistic$type <- "realistic"
simu.data <- rbind(simu.burst, simu.spaced, simu.realistic)
simu.data$expe <- "simu"

emu.burst.makespan <- c()
emu.burst.mwt <- c()
emu.spaced.makespan <- c()
emu.spaced.mwt <- c()
emu.realistic.makespan <- c()
emu.realistic.mwt <- c()
for (i in 0:9) {
	met <- compute.metrics(paste("../results/200_delay170_", i, "_jobs.csv", sep=""))
	emu.burst.makespan <- c(emu.burst.makespan, met$makespan)
	emu.burst.mwt <- c(emu.burst.mwt, met$mwt)

	met <- compute.metrics(paste("../results/spaced_200_delay170_", i, "_jobs.csv", sep=""))
	emu.spaced.makespan <- c(emu.spaced.makespan, met$makespan)
	emu.spaced.mwt <- c(emu.spaced.mwt, met$mwt)
}
met <- compute.metrics("../results/KIT_0_jobs.csv")
emu.realistic.makespan <- c(emu.realistic.makespan, met$makespan)
emu.realistic.mwt <- c(emu.realistic.mwt, met$mwt)
emu.burst <- data.frame(makespan=emu.burst.makespan, mwt=emu.burst.mwt)
emu.burst$type <- "burst"
emu.spaced <- data.frame(makespan=emu.spaced.makespan, mwt=emu.spaced.mwt)
emu.spaced$makespan <- emu.spaced.makespan
emu.spaced$type <- "spaced"
emu.realistic <- data.frame(makespan=emu.realistic.makespan, mwt=emu.realistic.mwt)
emu.realistic$makespan <- emu.realistic.makespan
emu.realistic$type <- "realistic"
emu.data <- rbind(emu.burst, emu.spaced, emu.realistic)
emu.data$expe <- "emu"

alldata <- rbind(emu.data, simu.data) %>%
	group_by(type, expe) %>%
	summarize(mean_makespan=mean(makespan),
		  sd_makespan=sd(makespan),
		  mean_mwt=mean(mwt),
		  sd_mwt=sd(mwt))
