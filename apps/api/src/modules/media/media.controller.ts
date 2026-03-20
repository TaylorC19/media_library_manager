import { Body, Controller, Post } from "@nestjs/common";
import type {
  CreateManualMediaRecordRequest,
  ManualMediaRecordResponse
} from "@media-library/types";
import { CreateManualMediaRecordDto } from "./dto/create-manual-media-record.dto";
import { MediaService } from "./media.service";

@Controller("media")
export class MediaController {
  constructor(private readonly mediaService: MediaService) {}

  @Post("manual")
  async createManualMediaRecord(
    @Body() body: CreateManualMediaRecordDto
  ): Promise<ManualMediaRecordResponse> {
    const mediaRecord = await this.mediaService.createManualRecord(
      body as unknown as CreateManualMediaRecordRequest
    );

    return { mediaRecord };
  }
}
