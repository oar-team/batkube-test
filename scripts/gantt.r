library(tidyverse)
library(viridis)

args <- commandArgs(trailingOnly=T)
if (length(args) == 0) {
	stop("Usage : Rscript gantt.r dir/prefix", call.=F)
}


outputDir <- paste(c(dirname(args[1]), "/"), collapse="")
prefix <- basename(args[1])

jobs <- paste(c(outputDir, prefix, "_jobs.csv"), collapse="")
workload <- read.csv(jobs)

chunk <- workload %>% separate_rows(allocated_resources, sep=" ")
chunk <- chunk %>%
	separate(allocated_resources, sep="-", into = c("psetmin", "psetmax"), fill="right") %>%
	mutate(psetmax = as.integer(psetmax), psetmin = as.integer(psetmin)) %>%
	mutate(psetmax = ifelse(is.na(psetmax), psetmin, psetmax))
chunk  %>%
  ggplot(aes( xmin=starting_time,
                  ymin=psetmin,
                  ymax=psetmax + 0.9,
                  xmax=finish_time,
                  fill=workload_name)) +
  # We add the rectangles
  geom_rect(color="black", size=0.5, alpha=0.5) +
  # We also add a label
  geom_text(aes(x=starting_time +(finish_time-starting_time)/2,
                y=psetmin+((psetmax-psetmin)/2)+0.5,
                label=paste(job_id, "")), alpha=1,check_overlap = TRUE) +
  scale_color_viridis(discrete=F) + scale_fill_viridis(discrete=T) +
  # We rename our labels
  ylab("resources") + xlab("time (in seconds)") + theme_bw() + theme(legend.position = "none")

ggsave(paste(c(outputDir, prefix, "_gantt.png"), collapse=""))
